package utils

import (
	"fmt"
	"html"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	htmlTagRe   = regexp.MustCompile(`<[^>]*>`)
	pOpenRe     = regexp.MustCompile(`(?i)<p>`)
	pCloseRe    = regexp.MustCompile(`(?i)</p>`)
	brRe        = regexp.MustCompile(`(?i)<br\s*/?>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func StripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = pOpenRe.ReplaceAllString(s, "\n")
	s = pCloseRe.ReplaceAllString(s, "")
	s = brRe.ReplaceAllString(s, " ")
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}

func OpenInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

func FormatTimeAgo(unixTime int64) string {
	t := time.Unix(unixTime, 0)
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 30*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2, 2006")
}
