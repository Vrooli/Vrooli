package imessage

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"switchboard/internal/channels"
)

type Adapter struct {
	command  string
	lookPath func(string) (string, error)
	run      func(context.Context, string, string) ([]byte, error)
	interval time.Duration
}

func New() *Adapter { return NewWithCommand("osascript", exec.LookPath) }
func NewWithCommand(command string, lookPath func(string) (string, error)) *Adapter {
	if strings.TrimSpace(command) == "" {
		command = "osascript"
	}
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	return &Adapter{command: command, lookPath: lookPath, run: runScript, interval: 2 * time.Second}
}
func (a *Adapter) ID() string { return "imessage" }

func (a *Adapter) Connect(ctx context.Context, receive func(channels.Envelope) error) error {
	if reason := a.unavailableReason(); reason != "" {
		return errors.New(reason)
	}
	if receive == nil {
		return errors.New("iMessage receive callback is required")
	}
	lastReceived := time.Now().Add(-time.Second)
	for {
		output, err := a.run(ctx, lastReceived.Format(time.RFC3339Nano), a.command)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read iMessage events: %w", err)
		}
		for _, envelope := range parseEvents(string(output)) {
			if envelope.ReceivedAt.After(lastReceived) {
				lastReceived = envelope.ReceivedAt
			}
			if err := receive(envelope); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(a.interval):
		}
	}
}

func (a *Adapter) Send(ctx context.Context, out channels.Outbound) error {
	if reason := a.unavailableReason(); reason != "" {
		return errors.New(reason)
	}
	if strings.TrimSpace(out.ThreadKey) == "" {
		return errors.New("iMessage recipient is required")
	}
	script := `on run argv
tell application "Messages"
set targetService to 1st service whose service type is iMessage
set targetBuddy to buddy (item 1 of argv) of targetService
send (item 2 of argv) to targetBuddy
end tell
return "delivered"
end run`
	cmd := exec.CommandContext(ctx, a.command, "-e", script, "--", out.ThreadKey, out.Text)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (a *Adapter) Probe(context.Context) channels.ProbeResult {
	if reason := a.unavailableReason(); reason != "" {
		return channels.ProbeResult{Reason: reason}
	}
	return channels.ProbeResult{Available: true}
}

func (a *Adapter) unavailableReason() string {
	if runtime.GOOS != "darwin" {
		return "iMessage requires a macOS fleet node"
	}
	if _, err := a.lookPath(a.command); err != nil {
		return "osascript is not installed"
	}
	return ""
}

const receiveScript = `on run argv
set cutoff to date (item 1 of argv)
set outputLines to {}
tell application "Messages"
repeat with targetChat in chats
repeat with targetMessage in (messages of targetChat)
set receivedAt to date received of targetMessage
if receivedAt > cutoff then
set messageID to id of targetMessage as text
set chatID to id of targetChat as text
set senderAddress to "unknown"
try
set senderAddress to handle of sender of targetMessage as text
end try
set messageText to text of targetMessage
set end of outputLines to messageID & tab & chatID & tab & senderAddress & tab & (receivedAt as «class isot») & tab & messageText
end if
end repeat
end repeat
end tell
set AppleScript's text item delimiters to linefeed
return outputLines as text
end run`

func runScript(ctx context.Context, cutoff, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, "-e", receiveScript, "--", cutoff)
	return cmd.Output()
}

func parseEvents(output string) []channels.Envelope {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]channels.Envelope, 0, len(lines))
	for _, line := range lines {
		fields := strings.SplitN(line, "\t", 5)
		if len(fields) != 5 || strings.TrimSpace(fields[0]) == "" || strings.TrimSpace(fields[1]) == "" {
			continue
		}
		receivedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(fields[3]))
		if err != nil {
			continue
		}
		result = append(result, channels.Envelope{ChannelID: "imessage", RemoteMessageID: fields[0], ThreadKey: fields[1], SenderAddress: fields[2], AuthorKind: channels.AuthorHuman, Text: fields[4], ReceivedAt: receivedAt})
	}
	return result
}

var _ channels.Adapter = (*Adapter)(nil)
