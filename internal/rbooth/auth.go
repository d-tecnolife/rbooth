package rbooth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	adminSessionCookie = "rbooth_admin"
	guestAccessCookie  = "rbooth_guest"
	adminSessionTTL    = 12 * time.Hour
)

func (a *App) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.hasValidAdminSession(r) {
			next.ServeHTTP(w, r)
			return
		}

		target := "/admin/login?next=" + url.QueryEscape(sanitizeNextPath(r.URL.RequestURI()))
		http.Redirect(w, r, target, http.StatusSeeOther)
	})
}

func (a *App) requireGuest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.hasValidAdminSession(r) || a.hasValidGuestSession(r) {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimSpace(r.URL.Query().Get("access"))
		if token != "" && a.validateToken(token, "guest") {
			a.setCookie(w, r, guestAccessCookie, token, time.Time{})
			http.Redirect(w, r, stripAccessQuery(r.URL), http.StatusSeeOther)
			return
		}

		http.Error(w, "access requires the booth QR code", http.StatusForbidden)
	})
}

func (a *App) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if a.hasValidAdminSession(r) {
			http.Redirect(w, r, sanitizeNextPath(r.URL.Query().Get("next")), http.StatusSeeOther)
			return
		}

		a.render(w, "admin-login", pageData{
			Title:      "Admin Login",
			BaseURL:    a.baseURL,
			BoardURL:   a.boardURL(),
			CaptureURL: a.captureURL(),
			AdminURL:   a.adminURL(),
			Next:       sanitizeNextPath(r.URL.Query().Get("next")),
		})
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse login form", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	next := sanitizeNextPath(r.FormValue("next"))
	if subtle.ConstantTimeCompare([]byte(password), []byte(a.adminPassword)) != 1 {
		a.renderStatus(w, http.StatusUnauthorized, "admin-login", pageData{
			Title:      "Admin Login",
			BaseURL:    a.baseURL,
			BoardURL:   a.boardURL(),
			CaptureURL: a.captureURL(),
			AdminURL:   a.adminURL(),
			Next:       next,
			AuthError:  "Password was incorrect.",
		})
		return
	}

	expiresAt := time.Now().UTC().Add(adminSessionTTL)
	a.setCookie(w, r, adminSessionCookie, a.signToken("admin", expiresAt), expiresAt)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *App) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	a.clearCookie(w, r, adminSessionCookie)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) publicCaptureURL(r *http.Request) string {
	values := url.Values{}
	values.Set("access", a.signToken("guest", time.Time{}))
	return requestBaseURL(r, a.baseURL) + a.captureURL() + "?" + values.Encode()
}

func (a *App) hasValidAdminSession(r *http.Request) bool {
	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return false
	}
	return a.validateToken(cookie.Value, "admin")
}

func (a *App) hasValidGuestSession(r *http.Request) bool {
	cookie, err := r.Cookie(guestAccessCookie)
	if err != nil {
		return false
	}
	return a.validateToken(cookie.Value, "guest")
}

func (a *App) signToken(scope string, expiresAt time.Time) string {
	expiresUnix := int64(0)
	if !expiresAt.IsZero() {
		expiresUnix = expiresAt.UTC().Unix()
	}

	payload := scope + "." + strconv.FormatInt(expiresUnix, 10)
	mac := hmac.New(sha256.New, a.authSecret)
	_, _ = mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + signature
}

func (a *App) validateToken(token string, expectedScope string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	scope := parts[0]
	if subtle.ConstantTimeCompare([]byte(scope), []byte(expectedScope)) != 1 {
		return false
	}

	expiresUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	if expiresUnix > 0 && time.Now().UTC().Unix() > expiresUnix {
		return false
	}

	payload := scope + "." + parts[1]
	mac := hmac.New(sha256.New, a.authSecret)
	_, _ = mac.Write([]byte(payload))
	expectedSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expectedSignature)) == 1
}

func (a *App) setCookie(w http.ResponseWriter, r *http.Request, name, value string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
	}
	if !expiresAt.IsZero() {
		cookie.Expires = expiresAt
	}
	http.SetCookie(w, cookie)
}

func (a *App) clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Secure:   requestIsSecure(r),
	})
}

func sanitizeNextPath(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/admin"
	}
	return next
}

func stripAccessQuery(target *url.URL) string {
	copyURL := *target
	query := copyURL.Query()
	query.Del("access")
	copyURL.RawQuery = query.Encode()
	if copyURL.Path == "" {
		copyURL.Path = "/"
	}
	result := copyURL.String()
	if result == "" {
		return "/"
	}
	return result
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func requestBaseURL(r *http.Request, fallback string) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host != "" {
		return fmt.Sprintf("%s://%s", proto, host)
	}

	return strings.TrimRight(fallback, "/")
}
