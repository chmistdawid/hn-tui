package utils

import (
	"html"
	"os/exec"
	"regexp"
	"strings"
)

func StripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	s = html.UnescapeString(s)

	s = regexp.MustCompile(`(?i)<p>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(s, "")

	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, " ")

	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}

func OpenInBrowser(url string) error {
	cmd := exec.Command("xdg-open", url)
	return cmd.Start()
}
