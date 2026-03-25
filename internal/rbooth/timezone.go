package rbooth

import (
	"strings"
	"time"
)

const defaultDisplayTimeZone = "UTC"

func loadDisplayLocation(name string) (*time.Location, string) {
	resolvedName := strings.TrimSpace(name)
	if resolvedName == "" {
		resolvedName = defaultDisplayTimeZone
	}
	switch strings.ToUpper(resolvedName) {
	case "UTC", "UTC+0", "UTC+00:00":
		return time.UTC, "UTC"
	}
	location, err := time.LoadLocation(resolvedName)
	if err != nil {
		location, err = time.LoadLocation(defaultDisplayTimeZone)
		if err != nil {
			return time.UTC, "UTC"
		}
		return location, defaultDisplayTimeZone
	}
	return location, resolvedName
}

func formatDisplayTime(value time.Time, location *time.Location) string {
	return value.In(location).Format("15:04:05 MST")
}

func formatDisplayDateTime(value time.Time, location *time.Location) string {
	return value.In(location).Format("2006-01-02 15:04 MST")
}
