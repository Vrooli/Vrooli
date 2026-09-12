// Package mix overlays canonical PCM streams for experiment augmentation.
package mix

import (
	"encoding/binary"
	"fmt"
	"math"

	"audio-tools/internal/protoint"
)

const bytesPerSample = 2

const (
	minInt16 = -32768
	maxInt16 = 32767
)

// Stats describes the realized overlay operation.
type Stats struct {
	SignalRMS    float64
	OverlayRMS   float64
	OverlayGain  float64
	ActualSNRDB  float64
	ClippedCount int
}

// OverlayAtSNR overlays the second canonical PCM stream on the first and
// scales the overlay so signal:overlay RMS matches targetSNRDB. The output
// length is exactly len(signal); shorter overlays loop and longer overlays
// truncate.
func OverlayAtSNR(signal, overlay []byte, targetSNRDB float64) ([]byte, Stats, error) {
	if len(signal)%bytesPerSample != 0 || len(overlay)%bytesPerSample != 0 {
		return nil, Stats{}, fmt.Errorf("audio mix: PCM length must be even")
	}
	if len(signal) == 0 {
		return nil, Stats{}, fmt.Errorf("audio mix: signal is empty")
	}
	if len(overlay) == 0 {
		return nil, Stats{}, fmt.Errorf("audio mix: overlay is empty")
	}

	samples := len(signal) / bytesPerSample
	signalRMS := rms(signal, samples, false)
	overlayRMS := rms(loopBytes(overlay, len(signal)), samples, false)
	if overlayRMS == 0 {
		return nil, Stats{}, fmt.Errorf("audio mix: overlay RMS is zero")
	}
	targetOverlayRMS := signalRMS / math.Pow(10, targetSNRDB/20)
	if signalRMS == 0 {
		targetOverlayRMS = overlayRMS / math.Pow(10, targetSNRDB/20)
	}
	gain := targetOverlayRMS / overlayRMS

	out := make([]byte, len(signal))
	var overlayPower float64
	var clipped int
	for i := 0; i < samples; i++ {
		base := protoint.PCMInt16(binary.LittleEndian.Uint16(signal[i*2:]))
		rawOverlay := protoint.PCMInt16(binary.LittleEndian.Uint16(overlay[(i%(len(overlay)/2))*2:]))
		scaledOverlay := float64(rawOverlay) * gain
		overlayPower += scaledOverlay * scaledOverlay
		mixed := int(math.Round(float64(base) + scaledOverlay))
		switch {
		case mixed > maxInt16:
			mixed = maxInt16
			clipped++
		case mixed < minInt16:
			mixed = minInt16
			clipped++
		}
		binary.LittleEndian.PutUint16(out[i*2:], protoint.PCMUint16(int16(mixed)))
	}
	actualOverlayRMS := math.Sqrt(overlayPower / float64(samples))
	actualSNR := math.Inf(1)
	if actualOverlayRMS > 0 && signalRMS > 0 {
		actualSNR = 20 * math.Log10(signalRMS/actualOverlayRMS)
	}
	return out, Stats{
		SignalRMS:    signalRMS,
		OverlayRMS:   overlayRMS,
		OverlayGain:  gain,
		ActualSNRDB:  actualSNR,
		ClippedCount: clipped,
	}, nil
}

// RMS returns root-mean-square amplitude for canonical PCM bytes.
func RMS(pcm []byte) (float64, error) {
	if len(pcm)%bytesPerSample != 0 {
		return 0, fmt.Errorf("audio mix: PCM length must be even")
	}
	if len(pcm) == 0 {
		return 0, nil
	}
	return rms(pcm, len(pcm)/bytesPerSample, false), nil
}

func rms(pcm []byte, samples int, loop bool) float64 {
	if samples == 0 {
		return 0
	}
	var power float64
	available := len(pcm) / bytesPerSample
	for i := 0; i < samples; i++ {
		idx := i
		if loop {
			idx %= available
		}
		v := protoint.PCMInt16(binary.LittleEndian.Uint16(pcm[idx*2:]))
		power += float64(v) * float64(v)
	}
	return math.Sqrt(power / float64(samples))
}

func loopBytes(in []byte, targetLen int) []byte {
	if len(in) >= targetLen {
		return in[:targetLen]
	}
	out := make([]byte, 0, targetLen)
	for len(out) < targetLen {
		remaining := targetLen - len(out)
		if remaining >= len(in) {
			out = append(out, in...)
		} else {
			out = append(out, in[:remaining]...)
		}
	}
	return out
}
