// Package chains coordinates runtime reconfiguration across the three
// provider chains (STT/TTS/Summarize). Settings updates flow through
// Coordinator.Reconfigure so handler code never reaches into the
// individual chains.
package chains

import (
	"context"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
)

// Config captures the runtime-tunable subset of provider routing.
type Config struct {
	BYOKEnabled   bool
	VrooliEnabled bool
	LocalEnabled  bool
	TTLByOK       time.Duration
	TTLVrooli     time.Duration
}

// Probe is the union of per-chain Probe results.
type Probe struct {
	STT       sttchain.ProbeResult
	TTS       ttschain.ProbeResult
	Summarize summarizechain.ProbeResult
}

// Coordinator owns the three chains.
type Coordinator struct {
	STT       *sttchain.Chain
	TTS       *ttschain.Chain
	Summarize *summarizechain.Chain
}

// Reconfigure pushes the new config to every chain.
func (c *Coordinator) Reconfigure(cfg Config) {
	if c.STT != nil {
		c.STT.Reconfigure(cfg.BYOKEnabled, cfg.VrooliEnabled, cfg.LocalEnabled, cfg.TTLByOK, cfg.TTLVrooli)
	}
	if c.TTS != nil {
		c.TTS.Reconfigure(cfg.BYOKEnabled, cfg.VrooliEnabled, cfg.LocalEnabled, cfg.TTLByOK, cfg.TTLVrooli)
	}
	if c.Summarize != nil {
		c.Summarize.Reconfigure(cfg.BYOKEnabled, cfg.VrooliEnabled, cfg.LocalEnabled, cfg.TTLByOK, cfg.TTLVrooli)
	}
}

// Probe collects fresh availability across all chains.
func (c *Coordinator) Probe(ctx context.Context) Probe {
	var p Probe
	if c.STT != nil {
		p.STT = c.STT.Probe(ctx)
	}
	if c.TTS != nil {
		p.TTS = c.TTS.Probe(ctx)
	}
	if c.Summarize != nil {
		p.Summarize = c.Summarize.Probe(ctx)
	}
	return p
}
