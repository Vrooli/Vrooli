package providers

import (
	"context"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"storage-manager/internal/cleanup"
)

// RuntimeHomeProvider is the storage-manager adapter for regenerable runtime
// home entries declared by repo-contract. The contract supplies the lower
// safety bound; an operator policy may be more conservative, but cannot make
// a managed entry eligible earlier or above its declared byte ceiling.
type RuntimeHomeProvider struct {
	inner   *FileProvider
	maxAge  time.Duration
	maxByte int64
}

func NewRuntimeHomeProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) cleanup.Provider {
	return &RuntimeHomeProvider{inner: newFileProvider(files, clock, cfg, cleanup.SafetyTierRegenerable, cleanup.ProviderModeDisabled, cleanup.ApprovalModeNone, "runtime-home-remove", desktopPlatforms), maxAge: cfg.RetentionMaxAge, maxByte: cfg.RetentionMaxBytes}
}

// RuntimeHomeRetentionConfig carries the contract's parsed limits without
// exposing contract parsing to the generic file provider.
type RuntimeHomeRetentionConfig struct {
	MaxAge   time.Duration
	MaxBytes int64
}

func (p *RuntimeHomeProvider) Metadata() cleanup.ProviderMetadata { return p.inner.Metadata() }

func (p *RuntimeHomeProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	req.Policy = p.boundPolicy(req.Policy)
	return p.inner.Estimate(ctx, req)
}

func (p *RuntimeHomeProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	req.Policy = p.boundPolicy(req.Policy)
	return p.inner.Preview(ctx, req)
}

func (p *RuntimeHomeProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	return p.inner.Apply(ctx, req)
}

func (p *RuntimeHomeProvider) Verify(ctx context.Context, req cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return p.inner.Verify(ctx, req)
}

func (p *RuntimeHomeProvider) boundPolicy(in cleanup.ProviderPolicy) cleanup.ProviderPolicy {
	out := in
	if p.maxAge > 0 && !out.AllowFreshReclaim && (out.MinAge == 0 || out.MinAge < p.maxAge) {
		out.MinAge = p.maxAge
	}
	if p.maxByte > 0 && (out.MaxBytes == 0 || out.MaxBytes > p.maxByte) {
		out.MaxBytes = p.maxByte
	}
	return out
}

// parseRuntimeHomeRetention is intentionally small and fail-closed. A bad
// contract must prevent provider construction rather than silently broadening
// cleanup eligibility.
func parseRuntimeHomeRetention(maxAge, maxBytes string) (RuntimeHomeRetentionConfig, error) {
	var out RuntimeHomeRetentionConfig
	var err error
	if maxAge != "" {
		out.MaxAge, err = coreRetention.ParseAge(maxAge)
		if err != nil {
			return RuntimeHomeRetentionConfig{}, err
		}
	}
	if maxBytes != "" {
		out.MaxBytes, err = coreRetention.ParseBytes(maxBytes)
		if err != nil {
			return RuntimeHomeRetentionConfig{}, err
		}
	}
	return out, nil
}
