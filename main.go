package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zishida/failhook/handlers"
)

// FailHook manages the monitoring and failure handling
type FailHook struct {
	handlers []handlers.FailureHandler
	debug    bool
}

// NewFailHook creates a new FailHook instance
func NewFailHook(debug bool) *FailHook {
	return &FailHook{
		handlers: []handlers.FailureHandler{},
		debug:    debug,
	}
}

// AddHandler adds a failure handler to the FailHook
func (fh *FailHook) AddHandler(handler handlers.FailureHandler) {
	fh.handlers = append(fh.handlers, handler)
	if fh.debug {
		fmt.Printf("Added handler: %s\n", handler.Description())
	}
}

// RunCommand runs a command and captures its output and exit code
func (fh *FailHook) RunCommand(ctx context.Context, command string, args []string) (int, string, error) {
	if fh.debug {
		fmt.Printf("Running command: %s %s\n", command, strings.Join(args, " "))
	}

	startTime := time.Now()

	// Create a new command with the passed context
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	if fh.debug {
		fmt.Printf("Command completed in %v\n", duration)
	}

	var exitCode int
	if err != nil {
		// Try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			exitCode = 1
		}
	}

	output := strings.TrimSpace(stdout.String() + stderr.String())
	return exitCode, output, err
}

// HandleFailure executes all registered handlers with the exit code and output
func (fh *FailHook) HandleFailure(exitCode int, output string) {
	for _, handler := range fh.handlers {
		if fh.debug {
			fmt.Printf("Executing handler: %s\n", handler.Description())
		}
		if err := handler.Handle(exitCode, output); err != nil {
			fmt.Fprintf(os.Stderr, "Error with handler %s: %v\n", handler.Description(), err)
		}
	}
}

func main() {
	var (
		command      string
		webhook      string
		syslogMsg    string
		slackWebhook string
		slackMsg     string
		timeout      int
		debug        bool
		showUsage    bool
	)

	// Define custom flag set to deal with the "--" separator
	fs := flag.NewFlagSet("failhook", flag.ExitOnError)
	
	fs.StringVar(&command, "c", "", "Command to execute on failure")
	fs.StringVar(&webhook, "w", "", "Webhook URL to call on failure")
	fs.StringVar(&syslogMsg, "s", "", "Message to send to syslog on failure")
	fs.StringVar(&slackWebhook, "slack-webhook", "", "Slack webhook URL")
	fs.StringVar(&slackMsg, "slack-msg", "Command failed with exit code __STATUS_CODE__\n```\n__OUTPUT__\n```", "Message to send to Slack")
	fs.IntVar(&timeout, "timeout", 0, "Timeout in seconds (0 means no timeout)")
	fs.BoolVar(&debug, "d", false, "Enable debug mode")
	fs.BoolVar(&showUsage, "h", false, "Show help")

	// Find where the "--" separator is
	sepIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sepIndex = i
			break
		}
	}

	// If no separator or it's the last argument, show usage
	if sepIndex == -1 || sepIndex == len(os.Args)-1 {
		printUsage()
		fmt.Println("Error: Monitored command must be specified after --")
		os.Exit(1)
	}

	// Parse flags before the separator
	if err := fs.Parse(os.Args[1:sepIndex]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	if showUsage {
		printUsage()
		os.Exit(0)
	}

	// Get monitored command and its arguments
	monitoredCmd := os.Args[sepIndex+1]
	var monitoredArgs []string
	if len(os.Args) > sepIndex+2 {
		monitoredArgs = os.Args[sepIndex+2:]
	}

	if debug {
		fmt.Printf("Monitoring command: %s %s\n", monitoredCmd, strings.Join(monitoredArgs, " "))
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	if timeout > 0 {
		var cancelTimeout context.CancelFunc
		ctx, cancelTimeout = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancelTimeout()
	}
	
	// Handle CTRL+C gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		if debug {
			fmt.Println("Received interrupt signal, canceling command...")
		}
		cancel()
	}()
	defer cancel()

	// Create FailHook instance
	failhook := NewFailHook(debug)

	// Register handlers based on flags
	if command != "" {
		failhook.AddHandler(handlers.NewCommandHandler(command))
	}
	if webhook != "" {
		failhook.AddHandler(handlers.NewWebhookHandler(webhook))
	}
	if syslogMsg != "" {
		failhook.AddHandler(handlers.NewSyslogHandler(syslogMsg))
	}
	if slackWebhook != "" {
		failhook.AddHandler(handlers.NewSlackHandler(slackWebhook, slackMsg))
	}

	// Run the monitored command
	exitCode, output, err := failhook.RunCommand(ctx, monitoredCmd, monitoredArgs)

	// Check if the context was canceled due to timeout
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintf(os.Stderr, "Command timed out after %d seconds\n", timeout)
		exitCode = 124 // Standard timeout exit code
		output = fmt.Sprintf("Command timed out after %d seconds", timeout)
	} else if ctx.Err() == context.Canceled && err != nil {
		fmt.Fprintf(os.Stderr, "Command was interrupted\n")
		exitCode = 130 // Standard exit code for SIGINT
		output = "Command was interrupted"
	}

	// If command succeeded, exit normally
	if exitCode == 0 {
		if debug {
			fmt.Println("Command succeeded, exiting normally")
		}
		os.Exit(0)
	}

	// If command failed, handle failure actions
	if debug {
		fmt.Printf("Command failed with exit code %d, executing handlers\n", exitCode)
	}
	failhook.HandleFailure(exitCode, output)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  failhook [OPTIONS] -- program [args...]")
	fmt.Println("\nOptions:")
	fmt.Println("  -c  Command to execute on failure")
	fmt.Println("  -w  Webhook URL to call on failure")
	fmt.Println("  -s  Message to send to syslog on failure")
	fmt.Println("  -slack-webhook  Slack webhook URL")
	fmt.Println("  -slack-msg      Message to send to Slack (default: \"Command failed with exit code __STATUS_CODE__\\n```\\n__OUTPUT__\\n```\")")
	fmt.Println("  -timeout        Timeout in seconds (0 means no timeout)")
	fmt.Println("  -d              Enable debug mode")
	fmt.Println("  -h              Show this help message")
	fmt.Println("\nPlaceholders:")
	fmt.Println("  __STATUS_CODE__  Exit code of the failed command")
	fmt.Println("  __OUTPUT__       Combined stdout and stderr output of the failed command")
	fmt.Println("  __TIMESTAMP__    Current timestamp in RFC3339 format")
	fmt.Println("  __DATE__         Current date (YYYY-MM-DD)")
	fmt.Println("  __TIME__         Current time (HH:MM:SS)")
	fmt.Println("\nExamples:")
	fmt.Println("  failhook -c \"echo 'Command failed with code: __STATUS_CODE__'\" -- /path/to/program")
	fmt.Println("  failhook -w \"https://example.com/hook?status=__STATUS_CODE__&output=__OUTPUT__\" -- /path/to/program")
	fmt.Println("  failhook -slack-webhook \"https://hooks.slack.com/services/XXX/YYY/ZZZ\" -- /path/to/program")
	fmt.Println("  failhook -timeout 30 -s \"Program timed out after 30s\" -- /path/to/long-running-program")
}