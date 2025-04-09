package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlackHandler(t *testing.T) {
	// Create a test server that validates the Slack message format
	var receivedPayload []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Read the request body
		payload := make([]byte, r.ContentLength)
		r.Body.Read(payload)
		receivedPayload = payload
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler with test server URL
	message := "Error: __STATUS_CODE__ - __OUTPUT__"
	handler := NewSlackHandler(server.URL, message, WithChannel("#testing"), WithUsername("TestBot"))

	// Execute the handler
	exitCode := 127
	output := "command not found"
	err := handler.Handle(exitCode, output)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	// Verify the Slack message format
	var slackMsg SlackMessage
	err = json.Unmarshal(receivedPayload, &slackMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal Slack message: %v", err)
	}

	// Check message text
	expectedText := strings.Replace(message, "__STATUS_CODE__", "127", -1)
	expectedText = strings.Replace(expectedText, "__OUTPUT__", "command not found", -1)
	if slackMsg.Text != expectedText {
		t.Errorf("message = %q, want %q", slackMsg.Text, expectedText)
	}

	// Check channel and username
	if slackMsg.Channel != "#testing" {
		t.Errorf("channel = %q, want %q", slackMsg.Channel, "#testing")
	}
	if slackMsg.Username != "TestBot" {
		t.Errorf("username = %q, want %q", slackMsg.Username, "TestBot")
	}

	// Test error response
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	errorHandler := NewSlackHandler(errorServer.URL, message)
	err = errorHandler.Handle(exitCode, output)
	if err == nil {
		t.Error("Expected error for 500 response, got nil")
	}

	// Test description
	desc := handler.Description()
	if !strings.Contains(desc, "Send to Slack") {
		t.Errorf("Description %q does not contain 'Send to Slack'", desc)
	}
}