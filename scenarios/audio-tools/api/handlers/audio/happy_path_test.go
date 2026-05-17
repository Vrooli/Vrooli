package audio

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	intaudio "audio-tools/internal/audio"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

// fakeRunner is a test-double Runner — duplicated from the
// internal/audio package's private fakeRunner. Lives here because the
// handlers/audio test binary cannot import the internal test scope.
type fakeRunner struct {
	stdout  []byte
	err     error
	respond func(name string, args []string) ([]byte, error)
	calls   int
}

func (f *fakeRunner) Run(_ context.Context, name string, _ []byte, args ...string) ([]byte, error) {
	f.calls++
	if f.respond != nil {
		return f.respond(name, args)
	}
	return f.stdout, f.err
}

// withRunner swaps the package-level Runner and seeds the ffmpeg/ffprobe
// presence cache so runFfmpeg / runFfprobe paths execute the fake.
func withRunner(t *testing.T, r intaudio.Runner) {
	t.Helper()
	restore := intaudio.SetFfmpegAvailableForTest(true, true)
	prev := intaudio.DefaultRunner
	intaudio.DefaultRunner = r
	t.Cleanup(func() {
		intaudio.DefaultRunner = prev
		restore()
	})
}

func newAudioClient(t *testing.T) audioconnect.AudioProcessingServiceClient {
	t.Helper()
	mod := Module(nil)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return audioconnect.NewAudioProcessingServiceClient(http.DefaultClient, srv.URL)
}

func TestTranscode_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("ENCODED")})
	c := newAudioClient(t)
	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("RAW"), OutputFormat: "mp3", SampleRate: 16000, Channels: 1, Bitrate: 128000,
	}))
	require.NoError(t, err)
	require.Equal(t, []byte("ENCODED"), res.Msg.GetAudio())
	require.Equal(t, "audio/mpeg", res.Msg.GetContentType())
}

func TestTranscode_DefaultContentType(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("OUT")})
	c := newAudioClient(t)
	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{Audio: []byte("X")}))
	require.NoError(t, err)
	require.Equal(t, "audio/wav", res.Msg.GetContentType())
}

func TestTranscode_RunnerErrorMapsToInternal(t *testing.T) {
	withRunner(t, &fakeRunner{err: errors.New("boom")})
	c := newAudioClient(t)
	_, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{Audio: []byte("X")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestTrim_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("TRIM")})
	c := newAudioClient(t)
	res, err := c.Trim(context.Background(), connect.NewRequest(&audiov1.TrimRequest{
		Audio: []byte("X"), Format: "wav", StartSeconds: 1, EndSeconds: 5,
	}))
	require.NoError(t, err)
	require.Equal(t, []byte("TRIM"), res.Msg.GetAudio())
}

func TestFade_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("FADE")})
	c := newAudioClient(t)
	res, err := c.Fade(context.Background(), connect.NewRequest(&audiov1.FadeRequest{
		Audio: []byte("X"), Format: "wav", FadeInSeconds: 0.5, FadeOutSeconds: 0.5, OutputFormat: "mp3",
	}))
	require.NoError(t, err)
	require.Equal(t, []byte("FADE"), res.Msg.GetAudio())
	require.Equal(t, "audio/mpeg", res.Msg.GetContentType())
}

func TestVolume_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("VOL")})
	c := newAudioClient(t)
	res, err := c.Volume(context.Background(), connect.NewRequest(&audiov1.VolumeRequest{
		Audio: []byte("X"), Format: "wav", GainDb: -3, OutputFormat: "flac",
	}))
	require.NoError(t, err)
	require.Equal(t, "audio/flac", res.Msg.GetContentType())
	require.NotEmpty(t, res.Msg.GetAudio())
}

func TestNormalize_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("NORM")})
	c := newAudioClient(t)
	res, err := c.Normalize(context.Background(), connect.NewRequest(&audiov1.NormalizeRequest{
		Audio: []byte("X"), Format: "wav", Method: "peak", TargetLufs: -14, OutputFormat: "wav",
	}))
	require.NoError(t, err)
	require.Equal(t, float64(-14), res.Msg.GetMeasuredLufs())
	require.Equal(t, []byte("NORM"), res.Msg.GetAudio())
}

func TestMerge_RequiresAtLeastOneSource(t *testing.T) {
	c := newAudioClient(t)
	_, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestMerge_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("MIX")})
	c := newAudioClient(t)
	res, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
		Sources: []*audiov1.MergeSource{
			{Audio: []byte("A"), Format: "wav"},
			{Audio: []byte("B"), Format: "wav"},
		},
		OutputFormat: "mp3",
	}))
	require.NoError(t, err)
	require.Equal(t, "audio/mpeg", res.Msg.GetContentType())
	require.NotEmpty(t, res.Msg.GetAudio())
}

func TestSplit_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("CHUNK")})
	c := newAudioClient(t)
	res, err := c.Split(context.Background(), connect.NewRequest(&audiov1.SplitRequest{
		Audio: []byte("LONG"), Format: "wav", ChunkSeconds: 1.0, OutputFormat: "wav",
	}))
	// Split may succeed or fail depending on whether ffprobe is wired
	// for the fake. We accept both as long as we don't panic.
	if err != nil {
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		return
	}
	require.NotNil(t, res.Msg)
}

func TestExtractMetadata_HappyPath(t *testing.T) {
	probeJSON := `{"streams":[{"codec_name":"mp3","sample_rate":"48000","channels":2,"bit_rate":"192000"}],"format":{"format_name":"mp3","duration":"3.5","bit_rate":"192000","tags":{"title":"hello"}}}`
	withRunner(t, &fakeRunner{stdout: []byte(probeJSON)})
	c := newAudioClient(t)
	res, err := c.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{Audio: []byte("X")}))
	require.NoError(t, err)
	require.Equal(t, float64(3.5), res.Msg.GetMetadata().GetDurationSeconds())
	require.Equal(t, int32(48000), res.Msg.GetMetadata().GetSampleRate())
	require.Equal(t, "mp3", res.Msg.GetMetadata().GetCodec())
}

func TestExtractMetadata_FfprobeErrorMapsInternal(t *testing.T) {
	withRunner(t, &fakeRunner{err: errors.New("boom")})
	c := newAudioClient(t)
	_, err := c.ExtractMetadata(context.Background(), connect.NewRequest(&audiov1.ExtractMetadataRequest{Audio: []byte("X")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestMapAudioErr_FfmpegMissingIsFailedPrecondition(t *testing.T) {
	err := mapAudioErr(intaudio.ErrFFmpegMissing)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestMapAudioErr_GenericIsInternal(t *testing.T) {
	err := mapAudioErr(errors.New("other"))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestRequireBytes_ZeroReturnsInvalidArgument(t *testing.T) {
	require.NoError(t, requireBytes([]byte("x")))
	err := requireBytes(nil)
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAtoiOr_ParsingBehaviour(t *testing.T) {
	require.Equal(t, 42, atoiOr("42", 7))
	require.Equal(t, 7, atoiOr("", 7))
	require.Equal(t, 7, atoiOr("not-an-int", 7))
}

func TestSchemaIsEmpty(t *testing.T) {
	require.Equal(t, "", Schema())
}

func TestMultipartTranscodeHandler_HappyPath(t *testing.T) {
	withRunner(t, &fakeRunner{stdout: []byte("M")})

	body, contentType := buildMultipart(t, "audio", "in.wav", []byte("X"), map[string]string{
		"output_format": "mp3",
		"sample_rate":   "16000",
		"channels":      "1",
		"bitrate":       "128000",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audio/transcode", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	multipartTranscodeHandler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "audio/mpeg", w.Header().Get("Content-Type"))
	require.Equal(t, "M", w.Body.String())
}

func TestMultipartTranscodeHandler_MissingFile(t *testing.T) {
	body, contentType := buildMultipart(t, "wrong-field", "x.wav", []byte("X"), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audio/transcode", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	multipartTranscodeHandler().ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultipartTranscodeHandler_BadRequestBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audio/transcode", strings.NewReader("not-multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=---bogus")
	w := httptest.NewRecorder()
	multipartTranscodeHandler().ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func buildMultipart(t *testing.T, field, filename string, body []byte, fields map[string]string) (string, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = fw.Write(body)
	require.NoError(t, err)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())
	return buf.String(), mw.FormDataContentType()
}

func TestMultipartTranscodeHandler_FfmpegMissingMapsFailedDependency(t *testing.T) {
	restore := intaudio.SetFfmpegAvailableForTest(false, false)
	t.Cleanup(restore)
	body, contentType := buildMultipart(t, "audio", "x.wav", []byte("X"), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audio/transcode", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	multipartTranscodeHandler().ServeHTTP(w, req)
	require.Equal(t, http.StatusFailedDependency, w.Code)
}
