package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"web-console/internal/audioports"
	inttts "web-console/internal/tts"
)

// newTTSAdapter wires package-main state into the internal TTS service. The
// behavior lives in api/internal/tts; this file only adapts the cross-domain
// callbacks until those callers move behind narrower interfaces too.
func newTTSAdapter(s *Server) inttts.HandlerService {
	deps := inttts.Deps{
		GetConfig:              s.getTTSConfig,
		SetConfig:              s.setTTSConfig,
		PersistConfig:          func(c inttts.Config) error { return inttts.SaveConfig(s.ttsConfigPath, c) },
		GetSummarizeConfig:     s.getTTSSummarizeConfig,
		SetSummarizeConfig:     s.setTTSSummarizeConfig,
		PersistSummarizeConfig: func(c inttts.SummarizeConfig) error { return inttts.SaveSummarizeConfig(s.ttsSummarizePath, c) },
		GetHookStatus:          s.getClaudeHookStatus,
		GetLastRouting:         func() (*inttts.AppendResult, time.Time) { return toTTSServiceRoutingSnapshot(s.getLastTTSRouting()) },
		GetRoutingBySource: func(source string) (*inttts.AppendResult, time.Time) {
			return toTTSServiceRoutingSnapshot(s.getLastTTSRoutingBySource(source))
		},
		GetLastAck: func() (*inttts.ClientAck, time.Time) { return toTTSServiceAckSnapshot(s.getLastTTSAck()) },
		GetAckBySource: func(source string) (*inttts.ClientAck, time.Time) {
			return toTTSServiceAckSnapshot(s.getLastTTSAckBySource(source))
		},
		GetLastPlaybackEvent: func() (*inttts.PlaybackEvent, time.Time) {
			return toTTSServicePlaybackSnapshot(s.getLastTTSPlaybackEvent())
		},
		RecordPlaybackEvent: func(ev inttts.PlaybackEvent) {
			s.recordTTSPlaybackEvent(TTSPlaybackEvent{
				Source:    ev.Source,
				Stage:     ev.Stage,
				Backend:   ev.Backend,
				SessionID: ev.SessionID,
				Message:   ev.Message,
			})
		},
		// TTS capability is owned by audio-tools; web-console reports the
		// audio-tools scenario status here so the legacy precondition gate
		// in internal/tts.Service.Synthesize stays meaningful. Transport
		// errors surface as typed Connect codes from the Remote* port.
		KokoroCapability: func(ctx context.Context) (string, string) {
			if s.capabilities == nil {
				return "available", "audio-tools (no capability registry)"
			}
			for _, cap := range s.capabilities.ResolveLiveness(ctx) {
				if cap.ID == "audio-tools" {
					label := strings.TrimSpace(cap.Message)
					if label == "" {
						label = "audio-tools"
					}
					return string(cap.Status), label
				}
			}
			return "available", "audio-tools (status unknown)"
		},
		GetCache: func(key inttts.CacheKey) (inttts.SynthesizeResult, bool) {
			if s.ttsPort == nil {
				return inttts.SynthesizeResult{}, false
			}
			out, ok := s.ttsPort.GetCached(context.Background(), audioports.CacheLookup{
				EventID: key.EventID,
				Voice:   key.Voice,
				Speed:   key.Speed,
				Version: key.Version,
			})
			if !ok {
				return inttts.SynthesizeResult{}, false
			}
			return inttts.SynthesizeResult{Audio: out.Audio, ContentType: out.ContentType}, true
		},
		PutCache: func(key inttts.CacheKey, audio []byte, contentType string) {
			// Cache writes for the synthesize-on-demand path still go through
			// the package-main cache field — it is the conversation-event-keyed
			// cache concern web-console owns (the dossier classifies it as glue).
			if s.ttsCache == nil {
				return
			}
			s.ttsCache.Put(key, audio, contentType)
		},
	}
	if s.ttsPort != nil {
		deps.SynthesizeAudio = func(ctx context.Context, in inttts.SynthesizeInput) (io.ReadCloser, string, error) {
			res, err := s.ttsPort.Synthesize(ctx, audioports.TTSRequest{
				Input:          in.Input,
				Voice:          in.Voice,
				ResponseFormat: in.ResponseFormat,
				Speed:          in.Speed,
			})
			if err != nil {
				return nil, "", err
			}
			return io.NopCloser(bytes.NewReader(res.Audio)), res.ContentType, nil
		}
		deps.ListVoiceCatalog = func(ctx context.Context) ([]inttts.Voice, error) {
			out, err := s.ttsPort.ListVoices(ctx)
			if err != nil {
				return nil, err
			}
			voices := make([]inttts.Voice, 0, len(out))
			for _, v := range out {
				voices = append(voices, inttts.Voice{ID: v.ID, Name: v.Name})
			}
			return voices, nil
		}
	}
	return inttts.NewService(deps)
}

func toTTSServiceRoutingSnapshot(r *ConversationAppendResult, at time.Time) (*inttts.AppendResult, time.Time) {
	if r == nil {
		return nil, at
	}
	return &inttts.AppendResult{
		Appended:  r.Appended,
		Code:      r.Code,
		Reason:    r.Reason,
		Source:    r.Source,
		SessionID: r.SessionID,
		EventID:   r.EventID,
		Sequence:  r.Sequence,
		Duplicate: r.Duplicate,
	}, at
}

func toTTSServiceAckSnapshot(a *TTSClientAck, at time.Time) (*inttts.ClientAck, time.Time) {
	if a == nil {
		return nil, at
	}
	return &inttts.ClientAck{
		EventID:   a.EventID,
		Source:    a.Source,
		SessionID: a.SessionID,
		Stage:     a.Stage,
		Backend:   a.Backend,
		Message:   a.Message,
	}, at
}

func toTTSServicePlaybackSnapshot(p *TTSPlaybackEvent, at time.Time) (*inttts.PlaybackEvent, time.Time) {
	if p == nil {
		return nil, at
	}
	return &inttts.PlaybackEvent{
		Source:    p.Source,
		Stage:     p.Stage,
		Backend:   p.Backend,
		SessionID: p.SessionID,
		Message:   p.Message,
	}, at
}
