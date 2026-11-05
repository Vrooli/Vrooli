package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// The v1.0 Kokoro model retains the voice names exposed by the native TTS
// contract. The IDs are model speaker IDs, not array positions chosen
// by this adapter; they come from the upstream v1.0 speaker table.
var voices = []struct {
	ID  string
	SID int
}{
	{ID: "af_heart", SID: 3},
	{ID: "af_bella", SID: 2},
	{ID: "af_nicole", SID: 6},
	{ID: "af_sarah", SID: 9},
	{ID: "af_sky", SID: 10},
	{ID: "am_adam", SID: 11},
	{ID: "am_michael", SID: 16},
	{ID: "bf_emma", SID: 21},
	{ID: "bf_isabella", SID: 22},
	{ID: "bm_george", SID: 26},
	{ID: "bm_lewis", SID: 27},
}

type speechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float32 `json:"speed"`
}

type speechHandler struct {
	engine    TTS
	encoder   Encoder
	streaming StreamingSTT
	speaker   SpeakerRuntime
	mu        sync.Mutex
}

func newHandler(engine TTS) http.Handler {
	return newHandlerWithEncoder(engine, newFFmpegEncoder())
}

func newHandlerWithEncoder(engine TTS, encoder Encoder) http.Handler {
	return newHandlerWithEncoderAndStreaming(engine, encoder, nil)
}

func newHandlerWithEncoderAndStreaming(engine TTS, encoder Encoder, streaming StreamingSTT) http.Handler {
	return newHandlerWithEncoderStreamingAndSpeaker(engine, encoder, streaming, nil)
}

func newHandlerWithEncoderStreamingAndSpeaker(engine TTS, encoder Encoder, streaming StreamingSTT, speaker SpeakerRuntime) http.Handler {
	h := &speechHandler{engine: engine, encoder: encoder, streaming: streaming, speaker: speaker}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.health)
	mux.HandleFunc("/v1/audio/voices", h.listVoices)
	mux.HandleFunc("/v1/audio/speech", h.speech)
	mux.HandleFunc("/v1/stream", h.stream)
	if speaker != nil {
		mux.Handle("/v1/profiles", speaker)
		mux.Handle("/v1/profiles/", speaker)
		mux.Handle("/v1/verify", speaker)
		mux.Handle("/v1/extract", speaker)
		mux.Handle("/v1/info", speaker)
		mux.Handle("/ready", speaker)
	}
	return mux
}

func (h *speechHandler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	streamingLoaded := h.streaming != nil
	speakerLoaded := h.speaker != nil
	encoderReady := true
	if status, ok := h.encoder.(encoderReadiness); ok {
		encoderReady = status.Ready()
	}
	if !encoderReady {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	status := "ok"
	if !encoderReady {
		status = "degraded"
	}
	_, _ = fmt.Fprintf(w, `{"status":%q,"encoder_ready":%t,"streaming_model_loaded":%t,"speaker_model_loaded":%t}`, status, encoderReady, streamingLoaded, speakerLoaded)
}

func (h *speechHandler) listVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ids := make([]string, 0, len(voices))
	for _, voice := range voices {
		ids = append(ids, voice.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ids)
}

func (h *speechHandler) speech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req speechRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.error(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if strings.TrimSpace(req.Input) == "" {
		h.error(w, http.StatusBadRequest, "Missing required field: input")
		return
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "mp3"
	}
	sid, ok := voiceSID(req.Voice)
	if !ok {
		h.error(w, http.StatusBadRequest, "unknown voice")
		return
	}
	if req.Speed <= 0 {
		req.Speed = 1
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	audio, err := h.engine.Synthesize(r.Context(), req.Input, sid, req.Speed)
	if err != nil {
		h.error(w, http.StatusInternalServerError, err.Error())
		return
	}
	encoded, contentType, err := h.encoder.Encode(r.Context(), audio, req.ResponseFormat)
	if err != nil {
		h.error(w, http.StatusInternalServerError, "audio encoding failed")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprint(len(encoded)))
	_, _ = w.Write(encoded)
}

type streamStartRequest struct {
	Type       string `json:"type"`
	SampleRate int    `json:"sample_rate"`
	Language   string `json:"language"`
}

type streamControlMessage struct {
	Type string `json:"type"`
}

func (h *speechHandler) stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.streaming == nil {
		h.error(w, http.StatusServiceUnavailable, "native streaming STT is unavailable")
		return
	}
	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		h.error(w, http.StatusBadRequest, err.Error())
		return
	}
	defer conn.close()

	opcode, payload, err := conn.readMessage()
	if err != nil || opcode != 0x1 {
		_ = conn.writeJSON([]byte(`{"type":"error","message":"first websocket message must be a start control message"}`))
		return
	}
	var start streamStartRequest
	if err := json.Unmarshal(payload, &start); err != nil || start.Type != "start" || (start.SampleRate != 0 && start.SampleRate != streamingSampleRate) {
		_ = conn.writeJSON([]byte(`{"type":"error","message":"start requires sample_rate 16000"}`))
		return
	}
	stream, err := h.streaming.NewStream()
	if err != nil {
		_ = conn.writeJSON([]byte(fmt.Sprintf(`{"type":"error","message":%q}`, err.Error())))
		return
	}
	defer stream.Close()
	if err := conn.writeJSON([]byte(`{"type":"ready","sample_rate":16000,"encoding":"s16le","channels":1}`)); err != nil {
		return
	}

	processed := 0
	for {
		opcode, payload, err = conn.readMessage()
		if err != nil {
			return
		}
		switch opcode {
		case 0x2:
			events, err := stream.AcceptPCM(payload)
			if err != nil {
				streamError(conn, err)
				return
			}
			processed++
			if err := writeSTTEvents(conn, events); err != nil {
				return
			}
			if err := conn.writeJSON([]byte(fmt.Sprintf(`{"type":"processed","processed_batches":%d}`, processed))); err != nil {
				return
			}
		case 0x1:
			var control streamControlMessage
			if err := json.Unmarshal(payload, &control); err != nil {
				streamError(conn, fmt.Errorf("invalid control message"))
				return
			}
			if control.Type != "end" {
				streamError(conn, fmt.Errorf("unsupported control message %q", control.Type))
				return
			}
			events, err := stream.Finish()
			if err != nil {
				streamError(conn, err)
				return
			}
			if err := writeSTTEvents(conn, events); err != nil {
				return
			}
			_ = conn.writeJSON([]byte(`{"type":"done"}`))
			return
		default:
			streamError(conn, fmt.Errorf("expected binary PCM or text end message"))
			return
		}
	}
}

func writeSTTEvents(conn *wsConn, events []STTEvent) error {
	for _, event := range events {
		typeName := "partial"
		if event.Final {
			typeName = "segment"
		}
		payload, err := json.Marshal(struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			StartMS int64  `json:"start_ms,omitempty"`
			EndMS   int64  `json:"end_ms,omitempty"`
		}{Type: typeName, Text: event.Text, StartMS: event.StartSample * 1000 / streamingSampleRate, EndMS: event.EndSample * 1000 / streamingSampleRate})
		if err != nil {
			return err
		}
		if err := conn.writeJSON(payload); err != nil {
			return err
		}
	}
	return nil
}

func streamError(conn *wsConn, err error) {
	_ = conn.writeJSON([]byte(fmt.Sprintf(`{"type":"error","message":%q}`, err.Error())))
}

func voiceSID(id string) (int, bool) {
	if id == "" {
		id = "af_heart"
	}
	for _, voice := range voices {
		if voice.ID == id {
			return voice.SID, true
		}
	}
	return 0, false
}

func encodeWAV(audio Audio) ([]byte, error) {
	if audio.SampleRate <= 0 || len(audio.Samples) == 0 {
		return nil, fmt.Errorf("audio is empty")
	}
	dataSize := len(audio.Samples) * 2
	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1)
	binary.LittleEndian.PutUint16(buf[22:24], 1)
	binary.LittleEndian.PutUint32(buf[24:28], uint32(audio.SampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(audio.SampleRate*2))
	binary.LittleEndian.PutUint16(buf[32:34], 2)
	binary.LittleEndian.PutUint16(buf[34:36], 16)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	for i, sample := range audio.Samples {
		if sample > 1 {
			sample = 1
		} else if sample < -1 {
			sample = -1
		}
		value := int16(sample * 32767)
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(value))
	}
	return buf, nil
}

func (h *speechHandler) error(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": message})
}
