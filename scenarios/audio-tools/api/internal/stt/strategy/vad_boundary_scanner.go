package strategy

// vadBoundaryScanner owns the frame cursor and silence-run state for one
// OverlapAgree session. It is deliberately session-local: scanning a growing
// PCM buffer can resume at the first frame not previously examined.
type vadBoundaryScanner struct {
	frameBytes          int
	silenceFramesNeeded int
	frameMs             int
	silenceRMS          float64
	nextFrame           int
	silentFrames        int
	hasVoiced           bool
	loggedFirstVoiced   bool
	maxSilentRunSeen    int
}

// scan advances through complete frames and returns the first silence
// boundary, or -1 when no boundary is available yet. Callbacks preserve the
// strategy's diagnostic ordering without coupling this state machine to its
// logger or audio cursor.
func (s *vadBoundaryScanner) scan(pcm []byte, onVoiced func(rms float64, frameIndex int), onBoundary func(boundary, silentFrames int)) int {
	for s.nextFrame+s.frameBytes <= len(pcm) {
		rms := frameRMS(pcm[s.nextFrame : s.nextFrame+s.frameBytes])
		if rms < s.silenceRMS {
			s.silentFrames++
			if s.silentFrames > s.maxSilentRunSeen {
				s.maxSilentRunSeen = s.silentFrames
			}
		} else {
			if !s.loggedFirstVoiced {
				s.loggedFirstVoiced = true
				onVoiced(rms, s.nextFrame/s.frameBytes)
			}
			s.silentFrames = 0
			s.hasVoiced = true
		}
		s.nextFrame += s.frameBytes
		if s.hasVoiced && s.silentFrames >= s.silenceFramesNeeded {
			boundary := s.nextFrame
			onBoundary(boundary, s.silentFrames)
			s.silentFrames = 0
			s.hasVoiced = false
			return boundary
		}
	}
	return -1
}
