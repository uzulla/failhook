package handlers

import (
	"testing"
)

func TestPlaceholderRegistry(t *testing.T) {
	registry := NewPlaceholderRegistry()

	// Test default placeholders
	tests := []struct {
		name       string
		text       string
		exitCode   int
		output     string
		wantPrefix string
	}{
		{
			name:       "STATUS_CODE placeholder",
			text:       "Exit code: __STATUS_CODE__",
			exitCode:   42,
			output:     "some output",
			wantPrefix: "Exit code: 42",
		},
		{
			name:       "OUTPUT placeholder",
			text:       "Output: __OUTPUT__",
			exitCode:   1,
			output:     "error: file not found",
			wantPrefix: "Output: error: file not found",
		},
		{
			name:       "TIMESTAMP placeholder",
			text:       "Time: __TIMESTAMP__",
			exitCode:   0,
			output:     "",
			wantPrefix: "Time: ", // Just check prefix since timestamp will vary
		},
		{
			name:       "DATE placeholder",
			text:       "Date: __DATE__",
			exitCode:   0,
			output:     "",
			wantPrefix: "Date: ", // Just check prefix since date will vary
		},
		{
			name:       "TIME placeholder",
			text:       "Time: __TIME__",
			exitCode:   0,
			output:     "",
			wantPrefix: "Time: ", // Just check prefix since time will vary
		},
		{
			name:       "Multiple placeholders",
			text:       "Command failed with code __STATUS_CODE__ at __TIMESTAMP__: __OUTPUT__",
			exitCode:   127,
			output:     "command not found",
			wantPrefix: "Command failed with code 127 at ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.Replace(tt.text, tt.exitCode, tt.output)
			if result == tt.text {
				t.Errorf("Replace() did not replace placeholders in %q", tt.text)
			}
			if !HasPrefix(result, tt.wantPrefix) {
				t.Errorf("Replace() = %q, want prefix %q", result, tt.wantPrefix)
			}
		})
	}
}

func TestCustomPlaceholder(t *testing.T) {
	registry := NewPlaceholderRegistry()
	
	// Register a custom placeholder
	registry.Register("__CUSTOM__", func(exitCode int, output string) string {
		return "custom-value"
	})

	result := registry.Replace("Custom placeholder: __CUSTOM__", 0, "")
	expected := "Custom placeholder: custom-value"

	if result != expected {
		t.Errorf("Replace() = %q, want %q", result, expected)
	}
}

func TestURLEncoding(t *testing.T) {
	registry := NewPlaceholderRegistry()
	
	text := "https://example.com?output=__OUTPUT__"
	output := "Error & Warning"
	
	result := registry.ReplaceURLEncoded(text, 0, output)
	expected := "https://example.com?output=Error+%26+Warning"

	if result != expected {
		t.Errorf("ReplaceURLEncoded() = %q, want %q", result, expected)
	}
}

// HasPrefix is a helper function that checks if a string starts with a prefix
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}