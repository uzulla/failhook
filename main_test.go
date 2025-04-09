package main

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFailHook_RunCommand(t *testing.T) {
	failhook := NewFailHook(false)
	
	// Test successful command
	t.Run("successful command", func(t *testing.T) {
		ctx := context.Background()
		exitCode, output, err := failhook.RunCommand(ctx, "echo", []string{"hello"})
		
		if exitCode != 0 {
			t.Errorf("exitCode = %d, want 0", exitCode)
		}
		if output != "hello" {
			t.Errorf("output = %q, want %q", output, "hello")
		}
		if err != nil {
			t.Errorf("error = %v, want nil", err)
		}
	})
	
	// Test failed command
	t.Run("failed command", func(t *testing.T) {
		ctx := context.Background()
		exitCode, _, err := failhook.RunCommand(ctx, "sh", []string{"-c", "exit 42"})
		
		if exitCode != 42 {
			t.Errorf("exitCode = %d, want 42", exitCode)
		}
		if err == nil {
			t.Error("error = nil, want error")
		}
	})
	
	// Test timeout
	t.Run("command timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		_, _, err := failhook.RunCommand(ctx, "sleep", []string{"1"})
		
		if err == nil {
			t.Error("error = nil, want context deadline exceeded error")
		}
	})

	// Test command with stdout and stderr
	t.Run("command with stdout and stderr", func(t *testing.T) {
		ctx := context.Background()
		cmd := `echo "standard output"; echo "standard error" >&2`
		exitCode, output, err := failhook.RunCommand(ctx, "sh", []string{"-c", cmd})
		
		if exitCode != 0 {
			t.Errorf("exitCode = %d, want 0", exitCode)
		}
		if !strings.Contains(output, "standard output") || !strings.Contains(output, "standard error") {
			t.Errorf("output = %q, want both stdout and stderr", output)
		}
		if err != nil {
			t.Errorf("error = %v, want nil", err)
		}
	})
}

// testHandler is a helper type for testing
type testHandler struct {
	outputPath string
}

// Handle implements the FailureHandler interface
func (h *testHandler) Handle(exitCode int, output string) error {
	return os.WriteFile(h.outputPath, []byte(output), 0644)
}

// Description implements the FailureHandler interface
func (h *testHandler) Description() string {
	return "Test handler"
}

func TestAddHandler(t *testing.T) {
	failhook := NewFailHook(false)
	
	// Create a temporary file to test handling
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.txt")
	
	// Add the handler
	handler := &testHandler{outputPath: outputFile}
	failhook.AddHandler(handler)
	
	if len(failhook.handlers) != 1 {
		t.Errorf("handler count = %d, want 1", len(failhook.handlers))
	}
	
	// Test handling failure
	testOutput := "test failure output"
	failhook.HandleFailure(1, testOutput)
	
	// Check if the handler was executed
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	
	if string(content) != testOutput {
		t.Errorf("handler output = %q, want %q", string(content), testOutput)
	}
}

func TestHandleFailure(t *testing.T) {
	failhook := NewFailHook(false)
	
	// Create multiple test handlers
	tempDir := t.TempDir()
	handler1 := &testHandler{outputPath: filepath.Join(tempDir, "output1.txt")}
	handler2 := &testHandler{outputPath: filepath.Join(tempDir, "output2.txt")}
	
	failhook.AddHandler(handler1)
	failhook.AddHandler(handler2)
	
	// Test handling failure with multiple handlers
	testOutput := "multi handler test"
	exitCode := 99
	failhook.HandleFailure(exitCode, testOutput)
	
	// Check if both handlers were executed
	for i, path := range []string{handler1.outputPath, handler2.outputPath} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read output file %d: %v", i+1, err)
		}
		
		if string(content) != testOutput {
			t.Errorf("handler %d output = %q, want %q", i+1, string(content), testOutput)
		}
	}
}

func TestDebugMode(t *testing.T) {
	// Redirect stdout to capture debug output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Run with debug mode enabled
	failhook := NewFailHook(true)
	handler := &testHandler{outputPath: filepath.Join(t.TempDir(), "output.txt")}
	failhook.AddHandler(handler)
	
	// Close the write end and restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read captured output
	var buf strings.Builder
	io.Copy(&buf, r)
	debugOutput := buf.String()
	
	// Verify debug message was printed
	if !strings.Contains(debugOutput, "Added handler") {
		t.Errorf("Debug output doesn't contain 'Added handler', got: %s", debugOutput)
	}
}

// Test command line argument parsing with custom FlagSet
func TestCommandLineParsing(t *testing.T) {
	// Test that we can create the correct FlagSet
	fs := flag.NewFlagSet("failhook", flag.ContinueOnError)
	
	// Define test variables to store flag values
	var (
		command    string
		webhook    string
		timeout    int
		debug      bool
	)
	
	// Set up flags
	fs.StringVar(&command, "c", "", "Command")
	fs.StringVar(&webhook, "w", "", "Webhook")
	fs.IntVar(&timeout, "timeout", 0, "Timeout")
	fs.BoolVar(&debug, "d", false, "Debug")
	
	// Test parsing basic arguments
	testArgs := []string{"-c", "echo test", "-timeout", "30", "-d", "--", "ls", "-la"}
	sepIndex := -1
	for i, arg := range testArgs {
		if arg == "--" {
			sepIndex = i
			break
		}
	}
	
	if sepIndex == -1 {
		t.Fatal("No separator found")
	}
	
	// Parse only the arguments before the separator
	err := fs.Parse(testArgs[:sepIndex])
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}
	
	// Check that the flags were parsed correctly
	if command != "echo test" {
		t.Errorf("command = %q, want %q", command, "echo test")
	}
	if timeout != 30 {
		t.Errorf("timeout = %d, want 30", timeout)
	}
	if !debug {
		t.Errorf("debug = %v, want true", debug)
	}
	
	// Check the monitored command and args
	if sepIndex+1 >= len(testArgs) {
		t.Fatal("No monitored command found")
	}
	
	monitoredCmd := testArgs[sepIndex+1]
	if monitoredCmd != "ls" {
		t.Errorf("monitoredCmd = %q, want %q", monitoredCmd, "ls")
	}
	
	var monitoredArgs []string
	if len(testArgs) > sepIndex+2 {
		monitoredArgs = testArgs[sepIndex+2:]
	}
	
	if len(monitoredArgs) != 1 || monitoredArgs[0] != "-la" {
		t.Errorf("monitoredArgs = %v, want %v", monitoredArgs, []string{"-la"})
	}
}