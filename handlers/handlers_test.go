package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCommandHandler(t *testing.T) {
	// Create a temporary file to store command output
	tmpFile, err := os.CreateTemp("", "command_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test command that writes output to the temp file
	handler := NewCommandHandler(fmt.Sprintf("echo 'Exit code: __STATUS_CODE__, Output: __OUTPUT__' > %s", tmpFile.Name()))

	// Execute the handler
	exitCode := 42
	output := "test output"
	err = handler.Handle(exitCode, output)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	// Verify the command was executed correctly
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expectedContent := fmt.Sprintf("Exit code: %d, Output: %s\n", exitCode, output)
	if string(content) != expectedContent {
		t.Errorf("handler output = %q, want %q", string(content), expectedContent)
	}

	// Test description
	desc := handler.Description()
	if !strings.Contains(desc, "Execute command") {
		t.Errorf("Description %q does not contain 'Execute command'", desc)
	}
}

func TestWebhookHandler(t *testing.T) {
	// Create a test server
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create handler with test server URL
	webhookURL := server.URL + "?status=__STATUS_CODE__&output=__OUTPUT__"
	handler := NewWebhookHandler(webhookURL)

	// Execute the handler
	exitCode := 404
	output := "Not Found & Error"
	err := handler.Handle(exitCode, output)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	// Verify the webhook was called with correct parameters
	expectedParams := fmt.Sprintf("/?status=%d&output=%s", exitCode, "Not+Found+%26+Error")
	if receivedURL != expectedParams {
		t.Errorf("webhook parameters = %q, want %q", receivedURL, expectedParams)
	}

	// Test error response
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	errorHandler := NewWebhookHandler(errorServer.URL)
	err = errorHandler.Handle(exitCode, output)
	if err == nil {
		t.Error("Expected error for 500 response, got nil")
	}

	// Test description
	desc := handler.Description()
	if !strings.Contains(desc, "Call webhook") {
		t.Errorf("Description %q does not contain 'Call webhook'", desc)
	}
}

func TestSyslogHandler(t *testing.T) {
	// This test only checks that the handler description works,
	// as actual syslog interaction is difficult to test
	handler := NewSyslogHandler("Test message: __STATUS_CODE__")
	
	desc := handler.Description()
	if !strings.Contains(desc, "Send to syslog") {
		t.Errorf("Description %q does not contain 'Send to syslog'", desc)
	}
}