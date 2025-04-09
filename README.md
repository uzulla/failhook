# FailHook

A utility that runs actions when your command fails.

## Features

- Monitors command execution (does nothing on success)
- Runs various actions on failure:
  - Execute shell commands
  - Call webhooks
  - Send messages to syslog
  - Send notifications to Slack
- Easily extendable with custom handlers
- Timeout control for long-running commands
- Debug mode for troubleshooting
- Dynamic messages with placeholders

## Basic Usage

```
failhook [OPTIONS] -- command [args...]
```

Always specify the command to monitor after the `--` separator.

### Quick Example

Monitor a command and show "Failed!" if it fails:

```bash
# This command should always succeed
failhook -c "echo 'Failed'" -- ls /tmp

# But this one will fail, triggering the echo command
failhook -c "echo 'Failed'" -- ls /nonexistent-directory
```

## Installation

```bash
# Clone the repository
git clone https://github.com/zishida/failhook.git
cd failhook

# Build
go build -o failhook

# Install (optional)
sudo mv failhook /usr/local/bin/
```

### Options

- `-c "command"` - Shell command to run on failure
- `-w "url"` - Webhook URL to call on failure
- `-s "message"` - Message to send to syslog on failure
- `-slack-webhook "url"` - Slack webhook URL for failure notifications
- `-slack-msg "message"` - Message to send to Slack (default: "Command failed with exit code __STATUS_CODE__\n```\n__OUTPUT__\n```")
- `-timeout N` - Set timeout in seconds for the monitored command (0 means no timeout)
- `-d` - Enable debug mode
- `-h` - Show help message

### Placeholders

You can use these placeholders in your commands, webhook URLs, and messages:

| Placeholder | Description |
|-------------|-------------|
| `__STATUS_CODE__` | Exit code of the failed command |
| `__OUTPUT__` | Combined stdout and stderr output (URL-encoded in webhooks) |
| `__TIMESTAMP__` | Current timestamp in RFC3339 format |
| `__DATE__` | Current date (YYYY-MM-DD) |
| `__TIME__` | Current time (HH:MM:SS) |

## Examples

### Run a command on failure

```bash
# Display error message on failure
failhook -c "echo 'Command failed with code __STATUS_CODE__'" -- /path/to/program arg1 arg2

# Run a Slack notification script on failure
failhook -c "/usr/local/bin/notify-slack.sh '__STATUS_CODE__' '__OUTPUT__'" -- /path/to/program
```

### Call a webhook on failure

```bash
# Send status and error message to webhook
failhook -w "https://example.com/webhook?status=__STATUS_CODE__&output=__OUTPUT__" -- /path/to/program

# Call webhook with timestamp
failhook -w "https://example.com/webhook?timestamp=__TIMESTAMP__&status=__STATUS_CODE__" -- /path/to/program
```

### Send to syslog on failure

```bash
# Basic syslog message
failhook -s "Program failed with code __STATUS_CODE__: __OUTPUT__" -- /path/to/program

# Syslog message with timestamp
failhook -s "[__DATE__ __TIME__] Program failed (code: __STATUS_CODE__)" -- /path/to/program
```

### Send to Slack on failure

```bash
# Send error message to Slack
failhook -slack-webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" -- /path/to/program

# Custom message
failhook -slack-webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
         -slack-msg "<!here> :warning: *Error:* Program failed with code __STATUS_CODE__\n```\n__OUTPUT__\n```" \
         -- /path/to/program
```

### Set a timeout

```bash
# 30 second timeout
failhook -timeout 30 -s "Program timed out after 30 seconds" -- /path/to/long-running-program

# Notify Slack on timeout
failhook -timeout 60 -slack-webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
         -slack-msg "Program timed out after 60 seconds" \
         -- /path/to/program
```

### Combine multiple actions

```bash
# Run command and notify Slack on failure
failhook -c "echo 'Command failed'" \
         -slack-webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
         -- /path/to/program

# Combine all actions
failhook -c "logger -t myapp 'Failure: __STATUS_CODE__'" \
         -w "https://example.com/webhook?status=__STATUS_CODE__" \
         -s "Syslog error: __OUTPUT__" \
         -slack-webhook "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
         -- /path/to/program
```

## Extending FailHook

FailHook is designed to be easily extendable.

### Adding Custom Handlers

Create your own failure handlers by implementing the `handlers.FailureHandler` interface:

```go
package main

import (
    "github.com/zishida/failhook/handlers"
)

// MyCustomHandler is a custom handler
type MyCustomHandler struct {
    // Handler fields
}

// Implement Handle method
func (h *MyCustomHandler) Handle(exitCode int, output string) error {
    // Implement failure action
    return nil
}

// Implement Description method
func (h *MyCustomHandler) Description() string {
    return "My custom handler"
}

// Register in main program
func main() {
    // ...
    failhook.AddHandler(&MyCustomHandler{})
    // ...
}
```

### Adding Custom Placeholders

Modify the `handlers/placeholder.go` file to add new placeholders:

```go
// Extend PlaceholderRegistry
registry := handlers.NewPlaceholderRegistry()

// Register custom placeholders
registry.Register("__HOSTNAME__", func(exitCode int, output string) string {
    hostname, _ := os.Hostname()
    return hostname
})

registry.Register("__RANDOM_ID__", func(exitCode int, output string) string {
    return fmt.Sprintf("%d", rand.Intn(1000))
})
```

## Developer Information

### Running Tests

```bash
go test ./...
```

### Project Structure

- `main.go` - Main program
- `handlers/` - Failure handler implementations
  - `handlers.go` - Basic handlers and interfaces
  - `slack.go` - Slack notification handler
  - `placeholder.go` - Placeholder processing system

## License

MIT License

Copyright (c) 2025 uzulla (zishida@gmail.com)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
