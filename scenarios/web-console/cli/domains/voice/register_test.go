package voice

import "testing"

func TestValidation(t *testing.T) {
	t.Run("transcribe_requires_audio_file", func(t *testing.T) {
		err := runTranscribe(nil, []string{})
		if err == nil || err.Error() != "--audio-file is required" {
			t.Fatalf("expected missing audio-file error, got %v", err)
		}
	})

	t.Run("speaker_enroll_requires_audio_file", func(t *testing.T) {
		err := runSpeakerEnroll(nil, []string{})
		if err == nil || err.Error() != "--audio-file is required" {
			t.Fatalf("expected missing audio-file error, got %v", err)
		}
	})
}
