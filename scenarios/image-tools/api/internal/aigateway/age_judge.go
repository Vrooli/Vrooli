package aigateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"image-tools/internal/safety"
)

// ageJudgePrompt is the question asked of the local vision model. It is also what
// scripts/minor-review-eval.py measures, so the two cannot drift: change both together.
const ageJudgePrompt = `You are the age check inside a child-safety filter. Look at the person in this image. Use everything visible: face, body proportions, skin, hands, build. If several people are visible, judge the youngest. If there is no face, judge from the body alone.
Answer with JSON only, exactly this shape:
{"apparent_age_years": <integer, your best estimate of the youngest person's age>, "could_be_under_18": <true or false>, "confidence": "high" | "medium" | "low"}`

// ageJudgeRole is the gateway role that serves the judge. The call is always local-only:
// an image that needed an age check is never sent to a hosted model.
const ageJudgeRole = "vision.default"

// ageJudgeTokens leaves room for a model that thinks before it answers.
const ageJudgeTokens = 1500

// Environment for the loopback fallback. The gateway's vision.default route is
// tried first; when it is unavailable (its breaker opens after provider execution
// errors), the same local model is asked directly on loopback. The image never
// leaves this machine either way, and the model name must match the local model
// the gateway role selects (docs/internal/SECURITY.md, "Age judge").
const (
	ageJudgeOllamaURLEnv = "IMAGE_TOOLS_OLLAMA_URL"
	ageJudgeModelEnv     = "IMAGE_TOOLS_AGE_JUDGE_MODEL"
	ageJudgeDefaultURL   = "http://127.0.0.1:11434"
	// The 4B vision model needs about 7.4 GB; on this shared 16 GB card, beside the
	// resident chat model and reranker, a cold load died ("unexpected EOF"). The 2B
	// sibling is the gateway role's first fallback, fits, and answered the measured
	// portrait as a confident adult.
	ageJudgeDefaultModel = "qwen3-vl:2b"
)

// visionAgeJudge answers the safety package's face-independent age question with
// the gateway's local vision role, falling back to the same Ollama model on
// loopback when the gateway route cannot execute.
type visionAgeJudge struct {
	gateway   VisionGateway
	ollamaURL string
	model     string
	client    *http.Client
}

// NewAgeJudge builds the second-opinion age judge on a vision gateway.
func NewAgeJudge(gateway VisionGateway) safety.AgeJudge {
	return &visionAgeJudge{
		gateway:   gateway,
		ollamaURL: envOr(ageJudgeOllamaURLEnv, ageJudgeDefaultURL),
		model:     envOr(ageJudgeModelEnv, ageJudgeDefaultModel),
		client:    &http.Client{Timeout: 10 * time.Minute},
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (j *visionAgeJudge) Judge(ctx context.Context, image []byte) (safety.AgeJudgement, error) {
	if j == nil {
		return safety.AgeJudgement{}, fmt.Errorf("age judge: not configured")
	}
	prepared, _, _, mediaType, err := prepareVisionBytes(image)
	if err != nil {
		return safety.AgeJudgement{}, err
	}
	var gatewayErr error
	if j.gateway != nil {
		answer, askErr := j.gateway.Ask(ctx, VisionRequest{
			Operation: "age_judge", Role: ageJudgeRole, Local: true, Prompt: ageJudgePrompt,
			Image: prepared, MediaType: mediaType, MaxOutputTokens: ageJudgeTokens,
		})
		if askErr == nil {
			return ParseAgeJudgement(answer.Text)
		}
		gatewayErr = askErr
	}
	text, ollamaErr := j.askOllama(ctx, prepared)
	if ollamaErr != nil {
		if gatewayErr != nil {
			return safety.AgeJudgement{}, fmt.Errorf("age judge gateway: %v; local Ollama: %w", gatewayErr, ollamaErr)
		}
		return safety.AgeJudgement{}, fmt.Errorf("age judge local Ollama: %w", ollamaErr)
	}
	return ParseAgeJudgement(text)
}

// askOllama asks the local Ollama server directly. It is a loopback call with one
// prepared image: no hosted provider, no outbound network, no API key.
func (j *visionAgeJudge) askOllama(ctx context.Context, image []byte) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":  j.model,
		"stream": false,
		"messages": []map[string]any{{
			"role":    "user",
			"content": ageJudgePrompt,
			"images":  []string{base64.StdEncoding.EncodeToString(image)},
		}},
		// The answer is a few dozen tokens. A larger budget lets a thinking model
		// spend it all on reasoning and return an empty content field.
		"options": map[string]any{"num_predict": 800},
	})
	if err != nil {
		return "", fmt.Errorf("encode Ollama request: %w", err)
	}
	url := strings.TrimRight(j.ollamaURL, "/") + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		var detail bytes.Buffer
		_, _ = detail.ReadFrom(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("Ollama returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(detail.String()))
	}
	var reply struct {
		Message struct {
			Content  string `json:"content"`
			Thinking string `json:"thinking"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
		return "", fmt.Errorf("decode Ollama reply: %w", err)
	}
	text := strings.TrimSpace(reply.Message.Content)
	if text == "" {
		// A thinking model can put a short JSON answer only in its reasoning field.
		text = strings.TrimSpace(reply.Message.Thinking)
	}
	if text == "" {
		return "", fmt.Errorf("Ollama returned no answer")
	}
	return text, nil
}

// ParseAgeJudgement reads the model's JSON reply. An unusable reply is an error, which
// the caller treats as "not cleared".
func ParseAgeJudgement(text string) (safety.AgeJudgement, error) {
	raw, err := extractJSON(completionText(text))
	if err != nil {
		return safety.AgeJudgement{}, fmt.Errorf("age judge reply: %w", err)
	}
	var reply struct {
		Age            float64 `json:"apparent_age_years"`
		CouldBeUnder18 *bool   `json:"could_be_under_18"`
		Confidence     string  `json:"confidence"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return safety.AgeJudgement{}, fmt.Errorf("age judge reply: %w", err)
	}
	if reply.CouldBeUnder18 == nil || reply.Age <= 0 {
		return safety.AgeJudgement{}, fmt.Errorf("age judge reply lacks an age or the under-18 flag: %s", clip(string(raw)))
	}
	judgement := safety.AgeJudgement{
		ApparentAge: int(reply.Age), CouldBeUnder18: *reply.CouldBeUnder18,
		Confidence: strings.ToLower(strings.TrimSpace(reply.Confidence)),
	}
	judgement.Detail = fmt.Sprintf("apparent age %d, under-18 flag %t, %s confidence", judgement.ApparentAge, judgement.CouldBeUnder18, judgement.Confidence)
	return judgement, nil
}

// mediaTypeOf labels a payload for the gateway (the reviewer already sniffed it once).
func mediaTypeOf(image []byte) string {
	switch {
	case len(image) >= 3 && image[0] == 0xFF && image[1] == 0xD8:
		return "image/jpeg"
	case len(image) >= 4 && string(image[:4]) == "RIFF":
		return "image/webp"
	default:
		return "image/png"
	}
}
