package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackHandler sends a message to Slack on failure
type SlackHandler struct {
	webhookURL string
	message    string
	channel    string
	username   string
	registry   *PlaceholderRegistry
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Text     string `json:"text"`
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username,omitempty"`
}

// NewSlackHandler creates a new SlackHandler with the specified webhook URL and message
func NewSlackHandler(webhookURL, message string, options ...func(*SlackHandler)) *SlackHandler {
	handler := &SlackHandler{
		webhookURL: webhookURL,
		message:    message,
		username:   "FailHook",
		registry:   NewPlaceholderRegistry(),
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

// WithChannel sets the Slack channel for the message
func WithChannel(channel string) func(*SlackHandler) {
	return func(h *SlackHandler) {
		h.channel = channel
	}
}

// WithUsername sets the username for the Slack message
func WithUsername(username string) func(*SlackHandler) {
	return func(h *SlackHandler) {
		h.username = username
	}
}

// Handle sends a message to Slack with placeholders replaced
func (h *SlackHandler) Handle(exitCode int, output string) error {
	// Replace placeholders
	message := h.registry.Replace(h.message, exitCode, output)

	// Create the Slack message payload
	slackMsg := SlackMessage{
		Text:     message,
		Username: h.username,
	}

	if h.channel != "" {
		slackMsg.Channel = h.channel
	}

	// Marshal the message to JSON
	payload, err := json.Marshal(slackMsg)
	if err != nil {
		return fmt.Errorf("error marshaling Slack message: %v", err)
	}

	// Post to Slack webhook
	resp, err := http.Post(h.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error sending Slack message: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status code %d", resp.StatusCode)
	}

	return nil
}

// Description returns a description of the handler
func (h *SlackHandler) Description() string {
	return fmt.Sprintf("Send to Slack: %s", h.message)
}