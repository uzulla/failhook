package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PlaceholderFunc defines a function that returns a string replacement for a placeholder
type PlaceholderFunc func(exitCode int, output string) string

// PlaceholderContext holds data about a command execution
type PlaceholderContext struct {
	ExitCode    int
	Output      string
	CommandName string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
}

// PlaceholderRegistry manages available placeholders
type PlaceholderRegistry struct {
	placeholders map[string]PlaceholderFunc
}

// NewPlaceholderRegistry creates a new registry with default placeholders
func NewPlaceholderRegistry() *PlaceholderRegistry {
	registry := &PlaceholderRegistry{
		placeholders: make(map[string]PlaceholderFunc),
	}

	// Register default placeholders
	registry.Register("__STATUS_CODE__", func(exitCode int, _ string) string {
		return fmt.Sprintf("%d", exitCode)
	})
	
	registry.Register("__OUTPUT__", func(_ int, output string) string {
		return output
	})
	
	registry.Register("__TIMESTAMP__", func(_ int, _ string) string {
		return time.Now().Format(time.RFC3339)
	})
	
	registry.Register("__DATE__", func(_ int, _ string) string {
		return time.Now().Format("2006-01-02")
	})
	
	registry.Register("__TIME__", func(_ int, _ string) string {
		return time.Now().Format("15:04:05")
	})

	return registry
}

// Register adds a new placeholder to the registry
func (pr *PlaceholderRegistry) Register(placeholder string, fn PlaceholderFunc) {
	pr.placeholders[placeholder] = fn
}

// Replace replaces all registered placeholders in the given text
func (pr *PlaceholderRegistry) Replace(text string, exitCode int, output string) string {
	result := text
	for placeholder, fn := range pr.placeholders {
		replacement := fn(exitCode, output)
		result = strings.Replace(result, placeholder, replacement, -1)
	}
	return result
}

// ReplaceURLEncoded replaces all registered placeholders in the given text and URL-encodes their values
func (pr *PlaceholderRegistry) ReplaceURLEncoded(text string, exitCode int, output string) string {
	result := text
	for placeholder, fn := range pr.placeholders {
		replacement := url.QueryEscape(fn(exitCode, output))
		result = strings.Replace(result, placeholder, replacement, -1)
	}
	return result
}