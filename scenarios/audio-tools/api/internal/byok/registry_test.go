package byok

import "testing"

func TestRegistriesContents(t *testing.T) {
	r := NewRegistries()
	if _, ok := r.STT["openai-whisper"]; !ok {
		t.Fatal("missing openai-whisper")
	}
	if _, ok := r.STT["deepgram"]; !ok {
		t.Fatal("missing deepgram")
	}
	if _, ok := r.TTS["openai-tts"]; !ok {
		t.Fatal("missing openai-tts")
	}
	if _, ok := r.TTS["elevenlabs"]; !ok {
		t.Fatal("missing elevenlabs")
	}
	if _, ok := r.Summarize["openrouter"]; !ok {
		t.Fatal("missing openrouter")
	}
}
