package composition

import (
	"context"
	"errors"
	"fmt"
	"math/big"
)

const maxPlannedFrames int64 = 1_000_000

// Rational is an exact numerator/denominator pair. Denominator must be > 0.
type Rational struct {
	Num int64 `json:"num"`
	Den int64 `json:"den"`
}

type RangePolicy uint8

const (
	RangeReject RangePolicy = iota
	RangeHoldFirst
	RangeHoldLast
)

// FrameClock describes the source timeline in PTS ticks and the requested
// output schedule. Trim is [TrimStartPTS, TrimEndPTS). PlaybackRate is the
// amount of source time consumed per second of output. SourcePTS must be
// monotonic and SourceEndPTS is the exclusive source EOF boundary.
type FrameClock struct {
	SourcePTS      []int64
	SourceTimeBase Rational // seconds per PTS tick
	SourceEndPTS   int64
	TrimStartPTS   int64
	TrimEndPTS     int64
	PlaybackRate   Rational
	OutputRate     Rational // frames per output second
	Underflow      RangePolicy
	EOF            RangePolicy
}

// FrameSample binds one output frame to the latest source PTS <= its exact
// target source time. No float timestamp or frame-index equivalence is used.
type FrameSample struct {
	OutputIndex       int64    `json:"output_index"`
	OutputTimeSeconds Rational `json:"output_time_seconds"`
	TargetSourcePTS   Rational `json:"target_source_pts"`
	SourceIndex       int      `json:"source_index"`
	SourcePTS         int64    `json:"source_pts"`
}

// PlanFrames creates a bounded rational output schedule. A source frame stays
// current until the next PTS; source times before the first frame or at/after
// SourceEndPTS require the corresponding explicit hold policy.
func PlanFrames(ctx context.Context, clock FrameClock) ([]FrameSample, error) {
	if ctx == nil {
		return nil, errors.New("clock context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(clock.SourcePTS) == 0 {
		return nil, errors.New("source PTS list is empty")
	}
	if !positive(clock.SourceTimeBase) || !positive(clock.PlaybackRate) || !positive(clock.OutputRate) {
		return nil, errors.New("time base, playback rate, and output rate must be positive rationals")
	}
	if (clock.Underflow != RangeReject && clock.Underflow != RangeHoldFirst) || (clock.EOF != RangeReject && clock.EOF != RangeHoldLast) {
		return nil, errors.New("unknown source range policy")
	}
	for i, pts := range clock.SourcePTS {
		if i > 0 && pts < clock.SourcePTS[i-1] {
			return nil, errors.New("source PTS values must be monotonic")
		}
	}
	if clock.SourceEndPTS <= clock.SourcePTS[len(clock.SourcePTS)-1] {
		return nil, errors.New("source EOF must be after the final source PTS")
	}
	if clock.TrimEndPTS <= clock.TrimStartPTS {
		return nil, errors.New("trim range must have positive duration")
	}
	delta := new(big.Int).Sub(big.NewInt(clock.TrimEndPTS), big.NewInt(clock.TrimStartPTS))
	outputDuration := new(big.Rat).SetInt(delta)
	outputDuration.Mul(outputDuration, new(big.Rat).SetFrac(big.NewInt(clock.SourceTimeBase.Num), big.NewInt(clock.SourceTimeBase.Den)))
	outputDuration.Mul(outputDuration, new(big.Rat).SetFrac(big.NewInt(clock.PlaybackRate.Den), big.NewInt(clock.PlaybackRate.Num)))
	outputDuration.Mul(outputDuration, new(big.Rat).SetFrac(big.NewInt(clock.OutputRate.Num), big.NewInt(clock.OutputRate.Den)))
	frameCountBig := ceilRat(outputDuration)
	if !frameCountBig.IsInt64() {
		return nil, errors.New("output frame count overflows int64")
	}
	frameCount := frameCountBig.Int64()
	if frameCount <= 0 || frameCount > maxPlannedFrames {
		return nil, fmt.Errorf("output frame count must be between 1 and %d", maxPlannedFrames)
	}
	result := make([]FrameSample, 0, frameCount)
	for i := int64(0); i < frameCount; i++ {
		if i&1023 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		outputTime := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(i), big.NewInt(clock.OutputRate.Den)), big.NewInt(clock.OutputRate.Num))
		sourceOffset := new(big.Rat).Set(outputTime)
		sourceOffset.Mul(sourceOffset, new(big.Rat).SetFrac(big.NewInt(clock.PlaybackRate.Num), big.NewInt(clock.PlaybackRate.Den)))
		sourceOffset.Quo(sourceOffset, new(big.Rat).SetFrac(big.NewInt(clock.SourceTimeBase.Num), big.NewInt(clock.SourceTimeBase.Den)))
		target := new(big.Rat).Add(new(big.Rat).SetInt64(clock.TrimStartPTS), sourceOffset)
		selectedIndex, selectedPTS, err := selectPTS(clock, target)
		if err != nil {
			return nil, fmt.Errorf("output frame %d: %w", i, err)
		}
		outTime, err := rationalFromBig(outputTime)
		if err != nil {
			return nil, fmt.Errorf("output frame %d time: %w", i, err)
		}
		targetPTS, err := rationalFromBig(target)
		if err != nil {
			return nil, fmt.Errorf("output frame %d source target: %w", i, err)
		}
		result = append(result, FrameSample{OutputIndex: i, OutputTimeSeconds: outTime, TargetSourcePTS: targetPTS, SourceIndex: selectedIndex, SourcePTS: selectedPTS})
	}
	return result, nil
}

func selectPTS(clock FrameClock, target *big.Rat) (int, int64, error) {
	first := clock.SourcePTS[0]
	end := new(big.Rat).SetInt64(clock.SourceEndPTS)
	if target.Cmp(end) >= 0 {
		if clock.EOF == RangeHoldLast {
			index := len(clock.SourcePTS) - 1
			return index, clock.SourcePTS[index], nil
		}
		return 0, 0, errors.New("target is at or beyond source EOF; explicit hold-last policy is required")
	}
	if target.Cmp(new(big.Rat).SetInt64(first)) < 0 {
		if clock.Underflow == RangeHoldFirst {
			return 0, first, nil
		}
		return 0, 0, errors.New("target precedes the first source PTS; explicit hold-first policy is required")
	}
	lo, hi := 0, len(clock.SourcePTS)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if new(big.Rat).SetInt64(clock.SourcePTS[mid]).Cmp(target) <= 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return 0, 0, errors.New("no source PTS at or before target")
	}
	return lo - 1, clock.SourcePTS[lo-1], nil
}

func positive(r Rational) bool { return r.Num > 0 && r.Den > 0 }

func ceilRat(r *big.Rat) *big.Int {
	n, d := r.Num(), r.Denom()
	q, rem := new(big.Int).QuoRem(n, d, new(big.Int))
	if rem.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

func rationalFromBig(r *big.Rat) (Rational, error) {
	if !r.Num().IsInt64() || !r.Denom().IsInt64() {
		return Rational{}, errors.New("rational exceeds int64 representation")
	}
	return Rational{Num: r.Num().Int64(), Den: r.Denom().Int64()}, nil
}
