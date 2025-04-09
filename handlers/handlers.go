package handlers

import (
	"fmt"
	"log/syslog"
	"net/http"
	"os"
	"os/exec"
)

// FailureHandler defines the interface for handling command failures
type FailureHandler interface {
	Handle(exitCode int, output string) error
	Description() string
}

// CommandHandler executes a shell command on failure
type CommandHandler struct {
	command   string
	registry *PlaceholderRegistry
}

// NewCommandHandler creates a new CommandHandler with the specified command
func NewCommandHandler(command string) *CommandHandler {
	return &CommandHandler{
		command:  command,
		registry: NewPlaceholderRegistry(),
	}
}

// Handle executes the shell command with placeholders replaced
func (h *CommandHandler) Handle(exitCode int, output string) error {
	// Replace placeholders
	command := h.registry.Replace(h.command, exitCode, output)

	// Execute shell
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Description returns a description of the handler
func (h *CommandHandler) Description() string {
	return fmt.Sprintf("Execute command: %s", h.command)
}

// WebhookHandler calls a webhook URL on failure
type WebhookHandler struct {
	webhookURL string
	registry   *PlaceholderRegistry
}

// NewWebhookHandler creates a new WebhookHandler with the specified URL
func NewWebhookHandler(webhookURL string) *WebhookHandler {
	return &WebhookHandler{
		webhookURL: webhookURL,
		registry:   NewPlaceholderRegistry(),
	}
}

// Handle calls the webhook URL with placeholders replaced
func (h *WebhookHandler) Handle(exitCode int, output string) error {
	// Replace placeholders with URL-encoded values
	webhookURL := h.registry.ReplaceURLEncoded(h.webhookURL, exitCode, output)

	// Make HTTP request
	resp, err := http.Get(webhookURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	}

	return nil
}

// Description returns a description of the handler
func (h *WebhookHandler) Description() string {
	return fmt.Sprintf("Call webhook: %s", h.webhookURL)
}

// SyslogHandler sends a message to syslog on failure
type SyslogHandler struct {
	message  string
	registry *PlaceholderRegistry
}

// NewSyslogHandler creates a new SyslogHandler with the specified message
func NewSyslogHandler(message string) *SyslogHandler {
	return &SyslogHandler{
		message:  message,
		registry: NewPlaceholderRegistry(),
	}
}

// Handle sends a message to syslog with placeholders replaced
func (h *SyslogHandler) Handle(exitCode int, output string) error {
	// Replace placeholders
	message := h.registry.Replace(h.message, exitCode, output)

	// Connect to syslog
	syslogWriter, err := syslog.New(syslog.LOG_ERR|syslog.LOG_USER, "failhook")
	if err != nil {
		return err
	}
	defer syslogWriter.Close()

	// Send message
	return syslogWriter.Err(message)
}

// Description returns a description of the handler
func (h *SyslogHandler) Description() string {
	return fmt.Sprintf("Send to syslog: %s", h.message)
}