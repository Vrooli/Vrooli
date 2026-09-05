//go:build !linux && !darwin && !windows

package procsampler

import "context"

// unsupportedSampler is the non-Linux fallback. There is no portable
// single-pass process table without a /proc filesystem, so Sample reports
// ErrUnsupported and callers degrade to no per-process attribution (the
// aggregate collectors continue to work via host inventory).
type unsupportedSampler struct{}

// NewSampler returns a sampler that always reports ErrUnsupported off Linux.
func NewSampler() Sampler { return unsupportedSampler{} }

func (unsupportedSampler) Sample(context.Context) ([]ProcessSample, error) {
	return nil, ErrUnsupported
}
