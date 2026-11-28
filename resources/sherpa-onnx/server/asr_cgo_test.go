//go:build cgo && sherpa_onnx

package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
)

// TestNativeStreamingModelSmoke is intentionally a short model-load/forward
// contract check. It is opt-in because model data is a managed resource
// artifact, not a repository fixture.
func TestNativeStreamingModelSmoke(t *testing.T) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_STREAMING_TEST_WAV")
	if modelDir == "" || wavPath == "" {
		t.Skip("set SHERPA_ONNX_STREAMING_MODEL_DIR and SHERPA_ONNX_STREAMING_TEST_WAV")
	}
	t.Setenv("SHERPA_ONNX_STREAMING_MODEL_DIR", modelDir)
	modelLoadStartedAt := time.Now()
	engine, err := newStreamingEngineFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	modelLoadDuration := time.Since(modelLoadStartedAt)
	defer engine.Close()
	stream, err := engine.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatal("test input is not a RIFF/WAVE file")
	}
	if binary.LittleEndian.Uint16(wav[34:36]) != 16 || binary.LittleEndian.Uint16(wav[22:24]) != 1 {
		t.Fatal("test input must be mono signed-16 PCM")
	}
	sampleRate := binary.LittleEndian.Uint32(wav[24:28])
	if sampleRate != streamingSampleRate {
		t.Fatalf("test input sample rate = %d Hz, want %d Hz", sampleRate, streamingSampleRate)
	}
	var text string
	var lastEndSample int64
	var firstTextAudioEndSample int64
	var firstTextWall time.Duration
	var firstTextObserved bool
	var modelFeedCallTime time.Duration
	pace := time.Duration(envInt("SHERPA_ONNX_STREAMING_PACE_MS", 0)) * time.Millisecond
	inferenceStartedAt := time.Now()
	for offset := 44; offset < len(wav); offset += 3200 {
		end := offset + 3200
		if end > len(wav) {
			end = len(wav)
		}
		if (end-offset)%2 != 0 {
			end--
		}
		if pace > 0 {
			streamEnd := time.Duration(int64((end-44)/2) * int64(time.Second) / int64(sampleRate))
			// Keep the requested packet cadence even when it is slower than
			// the 100 ms audio range in one native feed call.
			slots := (streamEnd + pace - 1) / pace
			if wait := time.Until(inferenceStartedAt.Add(slots * pace)); wait > 0 {
				time.Sleep(wait)
			}
		}
		feedStartedAt := time.Now()
		events, err := stream.AcceptPCM(wav[offset:end])
		modelFeedCallTime += time.Since(feedStartedAt)
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			if event.Text != "" {
				text = event.Text
				if !firstTextObserved {
					firstTextAudioEndSample = int64((end - 44) / 2)
					firstTextWall = time.Since(inferenceStartedAt)
					firstTextObserved = true
				}
			}
			if event.EndSample > lastEndSample {
				lastEndSample = event.EndSample
			}
		}
	}
	inferenceDuration := time.Since(inferenceStartedAt)
	events, err := stream.Finish()
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Text != "" {
			text = event.Text
		}
		if event.EndSample > lastEndSample {
			lastEndSample = event.EndSample
		}
	}
	if text == "" {
		t.Fatal("streaming model produced no text")
	}
	if maxClientSample := int64((len(wav) - 44) / 2); lastEndSample > maxClientSample {
		t.Fatalf("transcript end sample %d exceeds client audio length %d", lastEndSample, maxClientSample)
	}
	t.Logf("recognized text = %q", text)
	audioDuration := time.Duration(int64((len(wav)-44)/2) * int64(time.Second) / int64(sampleRate))
	rtf := 0.0
	if inferenceDuration > 0 {
		rtf = float64(audioDuration) / float64(inferenceDuration)
	}
	firstTextAudioDuration := time.Duration(firstTextAudioEndSample * int64(time.Second) / int64(sampleRate))
	t.Logf("model load = %s; first text observed before finish=%v after %s captured audio and %s wall time; model-feed call wall time = %s; stream elapsed = %s for %s audio (RTF %.2fx, pace=%s)",
		modelLoadDuration, firstTextObserved, firstTextAudioDuration, firstTextWall, modelFeedCallTime, inferenceDuration, audioDuration, rtf, pace)
	if expected := os.Getenv("SHERPA_ONNX_STREAMING_EXPECT_TEXT"); expected != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(expected)) {
		t.Fatalf("recognized text %q does not contain expected final text %q", text, expected)
	}
	assertExpectedFinalSuffix(t, text)
}

// TestNativeStreamingWebSocketSmoke covers the complete resource adapter with
// managed model artifacts. In particular, it verifies that the end control
// frame reaches nativeSTTStream.Finish (which supplies the bounded decoder
// tail) and that the resulting final segment survives websocket serialization.
func TestNativeStreamingWebSocketSmoke(t *testing.T) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_STREAMING_TEST_WAV")
	if modelDir == "" || wavPath == "" {
		t.Skip("set SHERPA_ONNX_STREAMING_MODEL_DIR and SHERPA_ONNX_STREAMING_TEST_WAV")
	}
	t.Setenv("SHERPA_ONNX_STREAMING_MODEL_DIR", modelDir)
	engine, err := newStreamingEngineFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" ||
		binary.LittleEndian.Uint16(wav[34:36]) != 16 || binary.LittleEndian.Uint16(wav[22:24]) != 1 ||
		binary.LittleEndian.Uint32(wav[24:28]) != streamingSampleRate {
		t.Fatal("test input must be mono signed-16 PCM WAV at 16 kHz")
	}

	server := httptest.NewServer(newHandlerWithEncoderAndStreaming(fakeTTS{}, fakeEncoder{}, engine))
	defer server.Close()
	addr := strings.TrimPrefix(server.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Minute)); err != nil {
		t.Fatalf("set websocket test deadline: %v", err)
	}
	reader := bufio.NewReader(conn)
	if _, err := fmt.Fprintf(conn, "GET /v1/stream HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n", addr); err != nil {
		t.Fatalf("write websocket handshake: %v", err)
	}
	if handshake, err := readUntilHeaderEnd(reader); err != nil || !strings.HasPrefix(handshake, "HTTP/1.1 101") {
		t.Fatalf("websocket handshake = %q, err = %v", handshake, err)
	}
	writeMaskedFrame(t, conn, 0x1, []byte(`{"type":"start","sample_rate":16000,"language":"en"}`))
	if ready := readJSONFrame(t, reader); !strings.Contains(ready, `"type":"ready"`) {
		t.Fatalf("ready = %s", ready)
	}
	var final strings.Builder
	var finalEndMS int64
	for offset := 44; offset < len(wav); offset += 3200 {
		end := offset + 3200
		if end > len(wav) {
			end = len(wav)
		}
		if (end-offset)%2 != 0 {
			end--
		}
		writeMaskedFrame(t, conn, 0x2, wav[offset:end])
		// Each accepted frame gets a processed acknowledgement after any
		// recognition events generated by that frame.
		for {
			message := readJSONFrame(t, reader)
			var event struct {
				Type  string `json:"type"`
				Text  string `json:"text"`
				EndMS int64  `json:"end_ms"`
			}
			if err := json.Unmarshal([]byte(message), &event); err != nil {
				t.Fatal(err)
			}
			if event.Type == "segment" {
				final.WriteString(event.Text)
				if event.EndMS > finalEndMS {
					finalEndMS = event.EndMS
				}
			}
			if strings.Contains(message, `"type":"processed"`) {
				break
			}
		}
	}
	writeMaskedFrame(t, conn, 0x1, []byte(`{"type":"end"}`))
	for {
		message := readJSONFrame(t, reader)
		var event struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			EndMS int64  `json:"end_ms"`
		}
		if err := json.Unmarshal([]byte(message), &event); err != nil {
			t.Fatal(err)
		}
		if event.Type == "segment" {
			final.WriteString(event.Text)
			if event.EndMS > finalEndMS {
				finalEndMS = event.EndMS
			}
		}
		if event.Type == "done" {
			break
		}
	}
	if expected := os.Getenv("SHERPA_ONNX_STREAMING_EXPECT_TEXT"); expected != "" && !strings.Contains(strings.ToLower(final.String()), strings.ToLower(expected)) {
		t.Fatalf("websocket final segments %q do not contain expected final text %q", final.String(), expected)
	}
	assertExpectedFinalSuffix(t, final.String())
	fileText := transcribeNativeWAV(t, engine, wav)
	if websocketWords, fileWords := lexicalWords(final.String()), lexicalWords(fileText); websocketWords != fileWords {
		t.Fatalf("websocket transcript %q does not match native file transcript %q (lexical forms %q != %q)",
			final.String(), fileText, websocketWords, fileWords)
	}
	if finalEndMS > int64((len(wav)-44)/2)*1000/streamingSampleRate {
		t.Fatalf("final segment end %d ms exceeds source audio duration", finalEndMS)
	}
	t.Logf("websocket final segments = %q", final.String())
}

// TestNativeStreamingParallelStreamsSmoke exercises the shared CPU recognizer
// at the resource's declared four-session limit with separate, independently
// ordered OnlineStreams. It is opt-in because the model and fixture are managed
// resource artifacts rather than repository data.
func TestNativeStreamingParallelStreamsSmoke(t *testing.T) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	wavPath := os.Getenv("SHERPA_ONNX_STREAMING_TEST_WAV")
	if modelDir == "" || wavPath == "" {
		t.Skip("set SHERPA_ONNX_STREAMING_MODEL_DIR and SHERPA_ONNX_STREAMING_TEST_WAV")
	}
	t.Setenv("SHERPA_ONNX_STREAMING_MODEL_DIR", modelDir)
	t.Setenv("SHERPA_ONNX_PROVIDER", "cpu")
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" ||
		binary.LittleEndian.Uint16(wav[34:36]) != 16 || binary.LittleEndian.Uint16(wav[22:24]) != 1 ||
		binary.LittleEndian.Uint32(wav[24:28]) != streamingSampleRate {
		t.Fatal("test input must be mono signed-16 PCM WAV at 16 kHz")
	}
	engine, err := newStreamingEngineFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	const streamCount = 4
	streams := make([]STTStream, streamCount)
	for i := range streams {
		streams[i], err = engine.NewStream()
		if err != nil {
			t.Fatalf("create stream %d: %v", i+1, err)
		}
		defer streams[i].Close()
	}
	type measurement struct {
		index             int
		firstAudio        time.Duration
		firstWall         time.Duration
		firstBeforeFinish bool
		finalText         string
		elapsed           time.Duration
		err               error
	}
	measured := make([]measurement, streamCount)
	results := make(chan measurement, streamCount)
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(streamCount)
	var base time.Time
	for i, stream := range streams {
		go func(index int, stream STTStream) {
			ready.Done()
			<-start
			result := measurement{index: index}
			for offset := 44; offset < len(wav); offset += 3200 {
				end := min(offset+3200, len(wav))
				if (end-offset)%2 != 0 {
					end--
				}
				streamEnd := time.Duration(int64((end-44)/2) * int64(time.Second) / streamingSampleRate)
				if wait := time.Until(base.Add(streamEnd)); wait > 0 {
					time.Sleep(wait)
				}
				events, acceptErr := stream.AcceptPCM(wav[offset:end])
				if acceptErr != nil {
					result.err = acceptErr
					results <- result
					return
				}
				for _, event := range events {
					if event.Text != "" && result.firstWall == 0 {
						result.firstAudio = streamEnd
						result.firstWall = time.Since(base)
						result.firstBeforeFinish = true
					}
					if event.Final && event.Text != "" {
						if result.finalText != "" {
							result.finalText += " "
						}
						result.finalText += event.Text
					}
				}
			}
			finalEvents, finishErr := stream.Finish()
			result.elapsed = time.Since(base)
			if finishErr != nil {
				result.err = finishErr
			} else {
				for _, event := range finalEvents {
					if event.Final && event.Text != "" {
						if result.finalText != "" {
							result.finalText += " "
						}
						result.finalText += event.Text
					}
				}
				if result.firstWall == 0 {
					result.firstWall = time.Since(base)
					result.firstAudio = time.Duration(int64((len(wav)-44)/2) * int64(time.Second) / streamingSampleRate)
				}
			}
			results <- result
		}(i, stream)
	}
	ready.Wait()
	base = time.Now()
	close(start)
	for range streamCount {
		result := <-results
		measured[result.index] = result
		if result.err != nil {
			t.Errorf("stream %d: %v", result.index+1, result.err)
			continue
		}
		if result.finalText == "" {
			t.Errorf("stream %d produced no final text", result.index+1)
		} else {
			assertExpectedFinalSuffix(t, result.finalText)
		}
		if !result.firstBeforeFinish {
			t.Errorf("stream %d did not emit text before Finish", result.index+1)
		}
		t.Logf("stream %d first text after %s captured audio / %s wall; completed in %s", result.index+1, result.firstAudio, result.firstWall, result.elapsed)
	}
	if measured[0].err != nil || measured[1].err != nil || measured[0].finalText == "" || measured[1].finalText == "" {
		return
	}
	fileText := transcribeNativeWAV(t, engine, wav)
	wantWords := lexicalWords(fileText)
	for _, result := range measured {
		if gotWords := lexicalWords(result.finalText); gotWords != wantWords {
			t.Errorf("stream %d lexical output %q does not match native direct-feed result %q", result.index+1, gotWords, wantWords)
		}
	}
	engine.Close()
	for i, stream := range streams {
		if _, err := stream.AcceptPCM(wav[44:46]); err == nil || !strings.Contains(err.Error(), "closed") {
			t.Errorf("stream %d after engine close: got error %v, want closed stream", i+1, err)
		}
		stream.Close()
	}
}

// TestNativeStreamingEngineCloseFencesActiveStreams ensures recognizer teardown
// waits for in-flight CPU stream work, marks every child stream closed, and
// does not deadlock when callers are still submitting frames.
func TestNativeStreamingEngineCloseFencesActiveStreams(t *testing.T) {
	modelDir := os.Getenv("SHERPA_ONNX_STREAMING_MODEL_DIR")
	if modelDir == "" {
		t.Skip("set SHERPA_ONNX_STREAMING_MODEL_DIR")
	}
	t.Setenv("SHERPA_ONNX_STREAMING_MODEL_DIR", modelDir)
	t.Setenv("SHERPA_ONNX_PROVIDER", "cpu")
	engine, err := newStreamingEngineFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	const streamCount = 4
	streams := make([]STTStream, streamCount)
	for i := range streams {
		streams[i], err = engine.NewStream()
		if err != nil {
			t.Fatalf("create stream %d: %v", i+1, err)
		}
	}
	pcm := make([]byte, 3200)
	progress := make(chan struct{}, streamCount)
	workerErrors := make(chan error, streamCount)
	var workers sync.WaitGroup
	for _, stream := range streams {
		workers.Add(1)
		go func(stream STTStream) {
			defer workers.Done()
			firstAccepted := false
			for {
				if _, acceptErr := stream.AcceptPCM(pcm); acceptErr != nil {
					if !strings.Contains(acceptErr.Error(), "closed") {
						workerErrors <- acceptErr
					}
					return
				}
				if !firstAccepted {
					firstAccepted = true
					progress <- struct{}{}
				}
			}
		}(stream)
	}
	for range streamCount {
		select {
		case <-progress:
		case err := <-workerErrors:
			t.Fatalf("stream failed before close: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("streams did not begin native processing")
		}
	}

	closed := make(chan struct{})
	go func() {
		engine.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(30 * time.Second):
		t.Fatal("engine close deadlocked behind active streams")
	}
	workers.Wait()
	close(workerErrors)
	for workerErr := range workerErrors {
		t.Errorf("active stream failed unexpectedly during engine close: %v", workerErr)
	}
	for i, stream := range streams {
		if _, err := stream.AcceptPCM(pcm); err == nil || !strings.Contains(err.Error(), "closed") {
			t.Errorf("stream %d after engine close: got error %v, want closed stream", i+1, err)
		}
		stream.Close()
	}
}

// transcribeNativeWAV runs the same fixture through the native file-feed path.
// Comparing lexical output with the WebSocket adapter catches lost or duplicated
// segments independently of a hand-picked suffix assertion.
func transcribeNativeWAV(t *testing.T, engine StreamingSTT, wav []byte) string {
	t.Helper()
	stream, err := engine.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var transcript strings.Builder
	appendFinal := func(events []STTEvent) {
		for _, event := range events {
			if event.Final && event.Text != "" {
				if transcript.Len() > 0 {
					transcript.WriteByte(' ')
				}
				transcript.WriteString(event.Text)
			}
		}
	}
	for offset := 44; offset < len(wav); offset += 3200 {
		end := min(offset+3200, len(wav))
		if (end-offset)%2 != 0 {
			end--
		}
		events, err := stream.AcceptPCM(wav[offset:end])
		if err != nil {
			t.Fatal(err)
		}
		appendFinal(events)
	}
	events, err := stream.Finish()
	if err != nil {
		t.Fatal(err)
	}
	appendFinal(events)
	return transcript.String()
}

func lexicalWords(text string) string {
	var words strings.Builder
	space := true
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			words.WriteRune(r)
			space = false
		} else if !space {
			words.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(words.String())
}

// assertExpectedFinalSuffix protects end-of-utterance flushing separately
// from the broader expected-text check. A transcript can contain its expected
// phrase early and still silently lose its final words at stream completion.
func assertExpectedFinalSuffix(t *testing.T, text string) {
	t.Helper()
	suffix := strings.TrimSpace(os.Getenv("SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX"))
	if suffix == "" {
		t.Fatal("set SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX to the expected final words so the smoke test can detect tail truncation")
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(text)), strings.ToLower(suffix)) {
		t.Fatalf("recognized text %q does not end with expected final suffix %q", text, suffix)
	}
}
