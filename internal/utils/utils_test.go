package utils

import (
	"strings"
	"testing"
	"time"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tag",
			input:    "<p>Hello world</p>",
			expected: "Hello world",
		},
		{
			name:     "multiple tags",
			input:    "<div><b>Bold</b> and <i>italic</i></div>",
			expected: "Bold and italic",
		},
		{
			name:     "nested tags",
			input:    "<p>This is <a href='http://example.com'>a link</a> test</p>",
			expected: "This is a link test",
		},
		{
			name:     "br tags",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "html entities",
			input:    "&lt;div&gt;Hello &amp; World&lt;/div&gt;",
			expected: "<div>Hello & World</div>",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace collapse",
			input:    "<p>  Multiple   spaces   here  </p>",
			expected: "Multiple spaces here",
		},
		{
			name:     "no tags",
			input:    "Just plain text",
			expected: "Just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripHTML_Benchmark(t *testing.T) {
	input := "<p>This is a <b>test</b> with <a href='http://example.com'>links</a> and <br/>breaks.</p>"
	for i := 0; i < 1000; i++ {
		StripHTML(input)
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		unixTime int64
		expected string
	}{
		{
			name:     "just now",
			unixTime: now.Add(-30 * time.Second).Unix(),
			expected: "just now",
		},
		{
			name:     "minutes ago",
			unixTime: now.Add(-5 * time.Minute).Unix(),
			expected: "5m ago",
		},
		{
			name:     "hours ago",
			unixTime: now.Add(-3 * time.Hour).Unix(),
			expected: "3h ago",
		},
		{
			name:     "days ago",
			unixTime: now.Add(-2 * 24 * time.Hour).Unix(),
			expected: "2d ago",
		},
		{
			name:     "old date",
			unixTime: time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC).Unix(),
			expected: "Jan 15, 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.unixTime)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo(%d) = %q, want %q", tt.unixTime, result, tt.expected)
			}
		})
	}
}

func TestStripHTML_LongText(t *testing.T) {
	// Test that StripHTML handles long text without issues
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		b.WriteString("<p>Paragraph ")
		b.WriteString(string(rune('0' + i%10)))
		b.WriteString(" with <b>bold</b> and <i>italic</i> text.</p>")
	}
	input := b.String()
	result := StripHTML(input)
	if result == "" {
		t.Error("StripHTML returned empty string for long input")
	}
	if strings.Contains(result, "<") {
		t.Error("StripHTML did not remove all HTML tags")
	}
}
