package rbooth

import (
	"fmt"
	"strings"
)

type Personalization struct {
	AppSubtitle      string
	HomeWelcomeTitle string
	HomeWelcomeBody  string
	HomeAccessTitle  string
	HomeAccessBody   string
	BoardEmptyTitle  string
	BoardEmptyBody   string
	AdminIntroTitle  string
	AdminIntroBody   string
}

func normalizePersonalization(appName string, value Personalization) Personalization {
	if strings.TrimSpace(value.AppSubtitle) == "" {
		value.AppSubtitle = "happy birthday"
	}
	if strings.TrimSpace(value.HomeWelcomeTitle) == "" {
		value.HomeWelcomeTitle = appName
	}
	if strings.TrimSpace(value.HomeWelcomeBody) == "" {
		value.HomeWelcomeBody = fmt.Sprintf("welcome to %s! enjoy taking pictures and posting them on the photo wall! the photo wall updates live, so you can see your pictures on the wall in real time.", appName)
	}
	if strings.TrimSpace(value.HomeAccessTitle) == "" {
		value.HomeAccessTitle = "use the qr code to enter."
	}
	if strings.TrimSpace(value.HomeAccessBody) == "" {
		value.HomeAccessBody = "the capture and photo wall pages only open after you scan the qr code."
	}
	if strings.TrimSpace(value.BoardEmptyTitle) == "" {
		value.BoardEmptyTitle = "the wall is ready"
	}
	if strings.TrimSpace(value.BoardEmptyBody) == "" {
		value.BoardEmptyBody = "new snapshots will pin themselves here and stack into the collage live."
	}
	if strings.TrimSpace(value.AdminIntroTitle) == "" {
		value.AdminIntroTitle = "keep the wall running."
	}
	if strings.TrimSpace(value.AdminIntroBody) == "" {
		value.AdminIntroBody = "use this page for qr access, wall links, and a quick status check while the photo wall stays on the main display."
	}
	return value
}
