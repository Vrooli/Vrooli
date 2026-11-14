package skillscripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// PacketSchema is packet.py's result schema (FILE-CONTRACT.md §2).
const PacketSchema = "research-packet-result/v1"

// PacketResult is packet.py's stdout for every subcommand. The desk reads it
// and never re-derives what it reports.
type PacketResult struct {
	Schema       string        `json:"schema"`
	OK           bool          `json:"ok"`
	CommissionID string        `json:"commission_id"`
	Folder       string        `json:"folder"`
	Written      bool          `json:"written"`
	Gates        []PacketGate  `json:"gates"`
	Cards        []PacketCard  `json:"cards"`
	Texts        *PacketTexts  `json:"texts"`
	Errors       []PacketError `json:"errors"`
	// ExitCode is the script's exit: 0 ok, 1 a gate failed, 2 invalid spec,
	// 3 filesystem error or folder exists.
	ExitCode int `json:"-"`
}

// PacketGate is one size or path gate. Value and Limit are numbers, strings
// or null, kept as JSON text.
type PacketGate struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
	Limit json.RawMessage `json:"limit"`
	OK    bool            `json:"ok"`
}

// PacketCard is one topic card and its workstream order.
type PacketCard struct {
	Topic       string   `json:"topic"`
	Mode        *string  `json:"mode"`
	Order       []string `json:"order"`
	OrderSource *string  `json:"order_source"`
	ReadFirst   *string  `json:"read_first"`
}

// PacketTexts are the rendered texts; only preview returns them.
type PacketTexts struct {
	Charter string `json:"charter"`
	Stance  string `json:"stance"`
	Kernel  string `json:"kernel"`
}

// PacketError is one refusal with packet.py's code.
type PacketError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Packets runs packet.py over one research home.
type Packets struct {
	Paths       Paths
	Runner      Runner
	ResearchDir string
}

// Preview renders a spec in memory and writes nothing.
func (p Packets) Preview(ctx context.Context, spec []byte) (*PacketResult, error) {
	return p.run(ctx, []string{"preview", "--home", p.ResearchDir, "--spec", "-"}, spec)
}

// Build renders, gates and writes a new commission folder.
func (p Packets) Build(ctx context.Context, spec []byte) (*PacketResult, error) {
	return p.run(ctx, []string{"build", "--home", p.ResearchDir, "--spec", "-"}, spec)
}

// Check gates an existing commission folder and writes nothing.
func (p Packets) Check(ctx context.Context, commissionID string) (*PacketResult, error) {
	return p.run(ctx, []string{"check", "--home", p.ResearchDir, commissionID}, nil)
}

func (p Packets) run(ctx context.Context, args []string, stdin []byte) (*PacketResult, error) {
	script := p.Paths.Packet()
	result, err := p.Runner.Run(ctx, script, args, stdin)
	if err != nil {
		return nil, err
	}
	if result.ExitCode < 0 || result.ExitCode > 3 {
		return nil, fmt.Errorf("packet.py %s exited %d: %s", args[0], result.ExitCode, firstLine(result.Stderr))
	}
	var out PacketResult
	if err := json.NewDecoder(bytes.NewReader(result.Stdout)).Decode(&out); err != nil {
		return nil, fmt.Errorf("packet.py %s printed no result (exit %d: %s): %w", args[0], result.ExitCode, firstLine(result.Stderr), err)
	}
	if out.Schema != PacketSchema {
		return nil, fmt.Errorf("packet.py printed schema %q; the desk reads %q", out.Schema, PacketSchema)
	}
	out.ExitCode = result.ExitCode
	return &out, nil
}
