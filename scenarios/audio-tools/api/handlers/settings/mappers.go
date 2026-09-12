package settings

import (
	"audio-tools/internal/byokstore"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

// toProto converts a ProviderConfig domain object to its proto wire form.
func toProto(c store.ProviderConfig) *settv1.ProviderConfig {
	return &settv1.ProviderConfig{
		ByokEnabled:           c.BYOKEnabled,
		VrooliEnabled:         c.VrooliEnabled,
		LocalEnabled:          c.LocalEnabled,
		WhisperUrl:            c.WhisperURL,
		KokoroUrl:             c.KokoroURL,
		OllamaUrl:             c.OllamaURL,
		LpbsBaseUrl:           c.LPBSBaseURL,
		LpbsAppBundleKey:      c.LPBSAppBundleKey,
		AvailTtlByokSeconds:   c.AvailTTLBYOKSeconds,
		AvailTtlVrooliSeconds: c.AvailTTLVrooliSecs,
	}
}

// credToProto converts a BYOK credential summary to its proto wire form.
func credToProto(c byokstore.Credential) *settv1.BYOKCredentialSummary {
	out := &settv1.BYOKCredentialSummary{
		ProviderId:  c.ProviderID,
		Capability:  c.Capability,
		Fingerprint: c.Fingerprint,
	}
	out.CreatedAt = protomap.TimeToProto(c.CreatedAt)
	if c.LastUsedAt != nil {
		out.LastUsedAt = protomap.TimeToProto(*c.LastUsedAt)
	}
	return out
}

// voiceOverridesToProto converts the list of voice override rows to
// their proto wire form.
func voiceOverridesToProto(rows []store.VoiceOverride) []*settv1.VoiceOverride {
	out := make([]*settv1.VoiceOverride, 0, len(rows))
	for _, v := range rows {
		out = append(out, &settv1.VoiceOverride{
			CanonicalVoice: v.CanonicalVoice,
			TierProvider:   v.TierProvider,
			AdapterVoice:   v.AdapterVoice,
		})
	}
	return out
}
