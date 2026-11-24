package stt

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (h *handlers) speakerStatusPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(_ cliapp.OperationContext) (*sttv1.SpeakerStatus, error) {
		resp, err := h.admin.GetSpeakerStatus(context.Background(), connect.NewRequest(&sttv1.GetSpeakerStatusRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get-speaker-status", err, nil)
		}
		if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
			return nil, fmt.Errorf("server returned no speaker status")
		}
		return resp.Msg.GetStatus(), nil
	}, renderSpeakerStatus, func(ctx cliapp.RunContext, st *sttv1.SpeakerStatus) error {
		return cliapp.PrintProtoJSON(ctx.Stdout(), st)
	})
}

func (h *handlers) speakerStatus(ctx cliapp.RunContext) error {
	return h.speakerStatusPrimitive().Run(ctx)
}

func renderSpeakerStatus(ctx cliapp.RunContext, st *sttv1.SpeakerStatus) error {
	cfg := st.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Speaker verification:\n")
	fmt.Fprintf(ctx.Stdout(), "  capability      = %s (%s)\n", st.GetCapability(), st.GetCapabilityLabel())
	fmt.Fprintf(ctx.Stdout(), "  resource_ready  = %t\n", st.GetResourceReady())
	fmt.Fprintf(ctx.Stdout(), "  enabled         = %t\n", cfg.GetEnabled())
	fmt.Fprintf(ctx.Stdout(), "  mode            = %s\n", speakerModeLabel(cfg.GetMode()))
	fmt.Fprintf(ctx.Stdout(), "  threshold       = %.2f\n", cfg.GetThreshold())
	fmt.Fprintf(ctx.Stdout(), "  reject_behavior = %s\n", rejectBehaviorLabel(cfg.GetRejectBehavior()))
	fmt.Fprintf(ctx.Stdout(), "  active_profiles = %v\n", cfg.GetProfileIds())
	fmt.Fprintf(ctx.Stdout(), "  extraction      = %t\n", cfg.GetExtractionEnabled())
	fmt.Fprintf(ctx.Stdout(), "  enrolled        = %d profile(s)\n", st.GetProfileCount())
	for _, p := range st.GetProfiles() {
		active := ""
		for _, id := range cfg.GetProfileIds() {
			if id == p.GetId() {
				active = " [active]"
				break
			}
		}
		fmt.Fprintf(ctx.Stdout(), "    - %s (%s, %d clip(s), %.1fs voiced)%s\n", p.GetId(), p.GetDisplayName(), p.GetClipCount(), p.GetTotalVoicedSeconds(), active)
	}
	if cfg.GetEnabled() && cfg.GetMode() != sttv1.SpeakerMode_SPEAKER_MODE_OFF {
		fmt.Fprintf(ctx.Stdout(), "  note: speaker isolation only protects the Whisper VAD path (per-segment\n")
		fmt.Fprintf(ctx.Stdout(), "        PCM). The Kyutai streaming engine emits no per-segment audio, so it\n")
		fmt.Fprintf(ctx.Stdout(), "        is NOT gated by speaker verification.\n")
	}
	return nil
}

// speakerConfig mutates the speaker-verification config. Only provided flags
// are changed. --profiles replaces the active binding list; --bind-profile
// appends one (reads the current list first).
func (h *handlers) speakerConfigPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*sttv1.UpdateSpeakerConfigResponse, error) {
		resp, err := h.updateSpeakerConfig(ctx)
		return resp, err
	}, renderSpeakerConfigUpdate, renderSpeakerConfigUpdate)
}

func (h *handlers) speakerConfig(ctx cliapp.RunContext) error {
	return h.speakerConfigPrimitive().Run(ctx)
}

func (h *handlers) updateSpeakerConfig(ctx cliapp.OperationContext) (*sttv1.UpdateSpeakerConfigResponse, error) {
	cfg, mask, err := h.speakerConfigUpdateFromFlags(ctx)
	if err != nil {
		return nil, err
	}
	if len(mask.Paths) == 0 {
		return nil, fmt.Errorf("at least one flag must be set (--mode, --threshold, --enabled, --profiles, --bind-profile, --reject-behavior, --fallback, --extraction-enabled, --min-decision-seconds, --score-smoothing)")
	}
	resp, err := h.admin.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: mask,
		Config:     cfg,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("update-speaker-config", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConfig() == nil {
		return nil, fmt.Errorf("server returned no speaker config")
	}
	return resp.Msg, nil
}

// speakerConfigUpdateFromFlags builds the config patch in the established flag
// validation order, including resolving --bind-profile against the current list.
func (h *handlers) speakerConfigUpdateFromFlags(ctx cliapp.OperationContext) (*sttv1.SpeakerConfig, *fieldmaskpb.FieldMask, error) {
	mask := &fieldmaskpb.FieldMask{}
	cfg := &sttv1.SpeakerConfig{}
	if err := applySpeakerModeThresholdEnabled(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	if err := applySpeakerBehaviorFlags(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	if err := applySpeakerTimingFlags(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	if err := h.applySpeakerProfileFlags(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	return cfg, mask, nil
}

func applySpeakerModeThresholdEnabled(ctx cliapp.OperationContext, cfg *sttv1.SpeakerConfig, mask *fieldmaskpb.FieldMask) error {
	if v := ctx.Flag("mode"); v != "" {
		m, err := speakerModeFromFlag(v)
		if err != nil {
			return err
		}
		cfg.Mode = m
		mask.Paths = append(mask.Paths, "mode")
	}
	if v := ctx.Flag("threshold"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--threshold must be a number: %q", v)
		}
		cfg.Threshold = f
		mask.Paths = append(mask.Paths, "threshold")
	}
	if err := applyBoolFlag(ctx, "enabled", "enabled", mask, func(value bool) { cfg.Enabled = value }); err != nil {
		return err
	}
	return nil
}

func applySpeakerBehaviorFlags(ctx cliapp.OperationContext, cfg *sttv1.SpeakerConfig, mask *fieldmaskpb.FieldMask) error {
	if v := ctx.Flag("reject-behavior"); v != "" {
		r, err := rejectBehaviorFromFlag(v)
		if err != nil {
			return err
		}
		cfg.RejectBehavior = r
		mask.Paths = append(mask.Paths, "reject_behavior")
	}
	if err := applyBoolFlag(ctx, "fallback", "fallback_without_verification", mask, func(value bool) { cfg.FallbackWithoutVerification = value }); err != nil {
		return err
	}
	if err := applyBoolFlag(ctx, "extraction-enabled", "extraction_enabled", mask, func(value bool) { cfg.ExtractionEnabled = value }); err != nil {
		return err
	}
	return nil
}

func applySpeakerTimingFlags(ctx cliapp.OperationContext, cfg *sttv1.SpeakerConfig, mask *fieldmaskpb.FieldMask) error {
	if v := ctx.Flag("min-decision-seconds"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--min-decision-seconds must be a number: %q", v)
		}
		cfg.MinDecisionSeconds = f
		mask.Paths = append(mask.Paths, "min_decision_seconds")
	}
	if v := ctx.Flag("score-smoothing"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--score-smoothing must be a number: %q", v)
		}
		cfg.ScoreSmoothing = f
		mask.Paths = append(mask.Paths, "score_smoothing")
	}
	return nil
}

func (h *handlers) applySpeakerProfileFlags(ctx cliapp.OperationContext, cfg *sttv1.SpeakerConfig, mask *fieldmaskpb.FieldMask) error {
	profilesFlag := ctx.Flag("profiles")
	bindFlag := ctx.Flag("bind-profile")
	if profilesFlag != "" && bindFlag != "" {
		return fmt.Errorf("use either --profiles (replace) or --bind-profile (append), not both")
	}
	if profilesFlag != "" {
		cfg.ProfileIds = splitCSV(profilesFlag)
		mask.Paths = append(mask.Paths, "profile_ids")
	}
	if bindFlag != "" {
		cur, err := h.admin.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("get-speaker-config", err, nil)
		}
		ids := append([]string{}, cur.Msg.GetConfig().GetProfileIds()...)
		already := false
		for _, id := range ids {
			if id == bindFlag {
				already = true
				break
			}
		}
		if !already {
			ids = append(ids, bindFlag)
		}
		cfg.ProfileIds = ids
		mask.Paths = append(mask.Paths, "profile_ids")
	}
	return nil
}

func renderSpeakerConfigUpdate(ctx cliapp.RunContext, resp *sttv1.UpdateSpeakerConfigResponse) error {
	out := resp.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved speaker config:\n")
	fmt.Fprintf(ctx.Stdout(), "  enabled         = %t\n", out.GetEnabled())
	fmt.Fprintf(ctx.Stdout(), "  mode            = %s\n", speakerModeLabel(out.GetMode()))
	fmt.Fprintf(ctx.Stdout(), "  threshold       = %.2f\n", out.GetThreshold())
	fmt.Fprintf(ctx.Stdout(), "  reject_behavior = %s\n", rejectBehaviorLabel(out.GetRejectBehavior()))
	fmt.Fprintf(ctx.Stdout(), "  active_profiles = %v\n", out.GetProfileIds())
	fmt.Fprintf(ctx.Stdout(), "  fallback        = %t\n", out.GetFallbackWithoutVerification())
	fmt.Fprintf(ctx.Stdout(), "  extraction      = %t\n", out.GetExtractionEnabled())
	fmt.Fprintf(ctx.Stdout(), "  min_decision_s  = %.1f\n", out.GetMinDecisionSeconds())
	fmt.Fprintf(ctx.Stdout(), "  score_smoothing = %.2f\n", out.GetScoreSmoothing())
	return nil
}

// speakerEnroll appends one or more enrollment clips to a voice profile. --file
// accepts a comma-separated list to enroll several clips (e.g. different devices
// or speaking styles) in one call — each becomes its own append against the same
// profile, strengthening the identity. Pass --activate to bind+enable in one
// step. A profile is one identity; each clip is one condition (--label).
type speakerEnrollmentResult struct {
	Path               string   `json:"path"`
	ProfileID          string   `json:"profile_id"`
	ClipID             string   `json:"clip_id"`
	Label              string   `json:"label"`
	VoicedSeconds      float64  `json:"voiced_seconds"`
	ClipCount          int32    `json:"clip_count"`
	TotalVoicedSeconds float64  `json:"total_voiced_seconds"`
	ModelName          string   `json:"model_name"`
	EmbeddingDim       int32    `json:"embedding_dim"`
	ActiveProfiles     []string `json:"active_profiles,omitempty"`
	Enabled            bool     `json:"enabled"`
	hasConfig          bool
}

type speakerEnrollmentBatch struct {
	Enrollments []speakerEnrollmentResult `json:"enrollments"`
}

func (h *handlers) speakerEnrollPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(ctx cliapp.OperationContext) (speakerEnrollmentBatch, error) {
		paths := splitCSV(ctx.Flag("file"))
		if len(paths) == 0 {
			return speakerEnrollmentBatch{}, fmt.Errorf("--file is required (comma-separated for multiple clips)")
		}

		var activate *bool
		if v := ctx.Flag("activate"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return speakerEnrollmentBatch{}, fmt.Errorf("--activate must be true|false: %q", v)
			}
			activate = boolPtr(b)
		}

		format := audioFormatFromFlag(ctx.Flag("format"))
		label := ctx.Flag("label")
		name := ctx.Flag("name")
		notes := ctx.Flag("notes")
		profileID := ctx.Flag("profile")

		result := speakerEnrollmentBatch{Enrollments: make([]speakerEnrollmentResult, 0, len(paths))}
		for i, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return speakerEnrollmentBatch{}, fmt.Errorf("read %s: %w", path, err)
			}
			req := &sttv1.EnrollSpeakerProfileRequest{
				Audio:       data,
				Format:      format,
				ProfileId:   profileID,
				DisplayName: name,
				Notes:       notes,
				Label:       label,
			}
			// Activate once, on the first clip; later clips just append.
			if i == 0 && activate != nil {
				req.AddToActive = activate
				req.Enable = activate
			}
			resp, err := h.admin.EnrollSpeakerProfile(context.Background(), connect.NewRequest(req))
			if err != nil {
				return speakerEnrollmentBatch{}, cliapp.WrapAPIError("enroll-speaker-profile", err, nil)
			}
			if resp == nil || resp.Msg == nil || resp.Msg.GetEnrollment() == nil {
				return speakerEnrollmentBatch{}, fmt.Errorf("server returned no speaker enrollment")
			}
			en := resp.Msg.GetEnrollment()
			// Subsequent clips land in the same profile (server may have generated id).
			profileID = en.GetProfileId()
			item := speakerEnrollmentResult{Path: path, ProfileID: en.GetProfileId(), ClipID: en.GetClipId(), Label: en.GetLabel(), VoicedSeconds: en.GetVoicedSeconds(), ClipCount: en.GetClipCount(), TotalVoicedSeconds: en.GetTotalVoicedSeconds(), ModelName: en.GetModelName(), EmbeddingDim: en.GetEmbeddingDim()}
			if cfg := resp.Msg.GetConfig(); cfg != nil {
				item.ActiveProfiles, item.Enabled = cfg.GetProfileIds(), cfg.GetEnabled()
				item.hasConfig = true
			}
			result.Enrollments = append(result.Enrollments, item)
		}
		return result, nil
	}, renderSpeakerEnrollment, renderSpeakerEnrollment)
}

func renderSpeakerEnrollment(ctx cliapp.RunContext, batch speakerEnrollmentBatch) error {
	fmt.Fprintf(ctx.Stdout(), "Enrolling %d clip(s):\n", len(batch.Enrollments))
	for i, en := range batch.Enrollments {
		fmt.Fprintf(ctx.Stdout(), "  [%d] %s -> clip %s (label=%q, %.1fs voiced) — %d clip(s) total\n", i+1, en.Path, en.ClipID, en.Label, en.VoicedSeconds, en.ClipCount)
	}
	if len(batch.Enrollments) > 0 {
		en := batch.Enrollments[len(batch.Enrollments)-1]
		fmt.Fprintf(ctx.Stdout(), "Profile %s: %d clip(s), %.1fs total voiced; model %s (dim %d)\n", en.ProfileID, en.ClipCount, en.TotalVoicedSeconds, en.ModelName, en.EmbeddingDim)
		if en.hasConfig {
			fmt.Fprintf(ctx.Stdout(), "  active_now = %v (enabled=%t)\n", en.ActiveProfiles, en.Enabled)
		}
	}
	return nil
}

// speakerClips lists the enrollment clips of a profile.
func (h *handlers) speakerClipsPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*sttv1.ListSpeakerProfileClipsResponse, error) {
		profile := ctx.Flag("profile")
		if profile == "" {
			return nil, fmt.Errorf("--profile is required")
		}
		resp, err := h.admin.ListSpeakerProfileClips(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfileClipsRequest{
			ProfileId: profile,
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list-speaker-profile-clips", err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("server returned no speaker clips")
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, response *sttv1.ListSpeakerProfileClipsResponse) error {
		fmt.Fprintf(ctx.Stdout(), "Profile %s: %d clip(s)\n", response.GetProfileId(), response.GetCount())
		for _, c := range response.GetClips() {
			fmt.Fprintf(ctx.Stdout(), "  - %s (label=%q, %.1fs voiced)\n", c.GetClipId(), c.GetLabel(), c.GetVoicedSeconds())
		}
		return nil
	}, nil)
}

func (h *handlers) speakerClips(ctx cliapp.RunContext) error {
	return h.speakerClipsPrimitive().Run(ctx)
}

// speakerDeleteClip removes one clip from a profile and recomputes its centroid.
// Deleting the last clip deletes the profile.
func (h *handlers) speakerDeleteClipPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*sttv1.DeleteSpeakerProfileClipResponse, error) {
		profile := ctx.Flag("profile")
		clip := ctx.Flag("clip")
		if profile == "" || clip == "" {
			return nil, fmt.Errorf("--profile and --clip are required")
		}
		resp, err := h.admin.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{
			ProfileId: profile,
			ClipId:    clip,
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete-speaker-profile-clip", err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("server returned no clip deletion result")
		}
		return resp.Msg, nil
	}, func(ctx cliapp.RunContext, m *sttv1.DeleteSpeakerProfileClipResponse) error {
		if m.GetDeletedProfile() {
			fmt.Fprintf(ctx.Stdout(), "Deleted clip %s — that was the last clip, so profile %s was removed.\n", m.GetClipId(), m.GetProfileId())
			return nil
		}
		fmt.Fprintf(ctx.Stdout(), "Deleted clip %s from profile %s — %d clip(s) remain (%.1fs total voiced).\n",
			m.GetClipId(), m.GetProfileId(), m.GetClipCount(), m.GetTotalVoicedSeconds())
		return nil
	}, nil)
}

func (h *handlers) speakerDeleteClip(ctx cliapp.RunContext) error {
	return h.speakerDeleteClipPrimitive().Run(ctx)
}
