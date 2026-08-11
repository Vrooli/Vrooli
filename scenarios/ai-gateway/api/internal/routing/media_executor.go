package routing

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
)

const mediaExecutionTimeout = 10 * time.Minute

type mediaExecutionIDKey struct{}

// ResourceOpenRouterMediaExecutor is the Gateway-owned media bridge. It calls
// the resource command, which owns the provider model policy, credentials,
// endpoint, and response transport. Gateway only validates the role and writes
// returned bytes to the caller-owned output reference.
type ResourceOpenRouterMediaExecutor struct {
	runner            providers.CommandRunner
	resolveCredential func(context.Context) (string, error)

	mu     sync.Mutex
	active map[string]context.CancelFunc
}

func NewResourceOpenRouterMediaExecutor(runner providers.CommandRunner) *ResourceOpenRouterMediaExecutor {
	if runner == nil {
		runner = providers.ExecRunner{}
	}
	return &ResourceOpenRouterMediaExecutor{
		runner:            runner,
		resolveCredential: resolveOpenRouterCredential,
		active:            make(map[string]context.CancelFunc),
	}
}

func (e *ResourceOpenRouterMediaExecutor) Execute(ctx context.Context, req *routingv1.SubmitMediaRequest) (*MediaExecutionResult, error) {
	if req == nil || req.GetRequest() == nil {
		return nil, errors.New("media request is required")
	}
	if req.GetRequest().GetKind() != sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION {
		return nil, fmt.Errorf("resource-openrouter media executor does not support %s", req.GetRequest().GetKind().String())
	}
	if req.GetOutputCount() != 1 {
		return nil, errors.New("resource-openrouter media executor requires output_count=1 because output_reference names one caller-owned file")
	}
	outputPath := strings.TrimSpace(req.GetOutputReference())
	if !filepath.IsAbs(outputPath) {
		return nil, fmt.Errorf("output_reference must be an absolute caller-owned path: %q", outputPath)
	}
	role := strings.TrimSpace(req.GetRequest().GetRole())
	if role == "" {
		role = "image.generate.default"
	}

	model, err := e.resolveModel(ctx, role)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	id, _ := ctx.Value(mediaExecutionIDKey{}).(string)
	if id != "" {
		e.mu.Lock()
		e.active[id] = cancel
		e.mu.Unlock()
	}
	defer func() {
		cancel()
		if id != "" {
			e.mu.Lock()
			delete(e.active, id)
			e.mu.Unlock()
		}
	}()

	args := []string{"images", "generate", "--role", role, "--output-count", "1"}
	if len(req.GetInputs()) > 0 {
		inputPath := strings.TrimSpace(req.GetInputs()[0].GetReference())
		if !filepath.IsAbs(inputPath) {
			return nil, fmt.Errorf("media input reference must be an absolute caller-owned path: %q", inputPath)
		}
		args = append(args, "--input-file", inputPath)
	}
	result, err := e.runResource(runCtx, providers.Command{
		Name:    "resource-openrouter",
		Args:    args,
		Stdin:   req.GetPrompt(),
		Timeout: mediaExecutionTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("resource-openrouter image generation: %w", err)
	}

	image, mediaType, err := decodeMediaImage([]byte(result.Stdout))
	if err != nil {
		return nil, err
	}
	if err := writeMediaOutput(outputPath, image); err != nil {
		return nil, err
	}
	checksum := sha256.Sum256(image)
	request := req.GetRequest()
	return &MediaExecutionResult{
		RouteEvidence: &routingv1.RouteEvidence{
			EventId:          newID("media-route"),
			RequestId:        request.GetRequestId(),
			Scenario:         request.GetScenario(),
			Operation:        request.GetOperation(),
			Role:             role,
			Profile:          request.GetProfile(),
			PrivacyClass:     request.GetPrivacyClass(),
			SelectedProvider: "resource-openrouter",
			SelectedLocality: "remote",
			SelectedModel:    model,
			Status:           "succeeded",
			PromptRedacted:   true,
			ResponseRedacted: true,
			CreatedAt:        nowUTC(),
			PolicyReasons:    []string{"resource-owned image role resolution"},
		},
		Outputs: []*routingv1.MediaOutput{{
			Reference: outputPath,
			MediaType: mediaType,
			Bytes:     int64(len(image)),
			Checksum:  "sha256:" + hex.EncodeToString(checksum[:]),
		}},
		ResolvedModel: model,
	}, nil
}

func (e *ResourceOpenRouterMediaExecutor) Cancel(_ context.Context, id string) error {
	e.mu.Lock()
	cancel := e.active[strings.TrimSpace(id)]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (e *ResourceOpenRouterMediaExecutor) resolveModel(ctx context.Context, role string) (string, error) {
	result, err := e.runResource(ctx, providers.Command{
		Name:    "resource-openrouter",
		Args:    []string{"policy", "resolve", "--role", role, "--json"},
		Timeout: providers.DefaultCommandTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("resolve media role %q: %w", role, err)
	}
	var response struct {
		Role  string `json:"role"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return "", fmt.Errorf("decode media role %q: %w", role, err)
	}
	if response.Role != role || strings.TrimSpace(response.Model) == "" {
		return "", fmt.Errorf("resource-openrouter returned no model for media role %q", role)
	}
	return response.Model, nil
}

func (e *ResourceOpenRouterMediaExecutor) runResource(ctx context.Context, command providers.Command) (providers.Result, error) {
	if e == nil || e.resolveCredential == nil {
		return providers.Result{}, errors.New("OpenRouter credential resolver is not configured")
	}
	credential, err := e.resolveCredential(ctx)
	if err != nil {
		return providers.Result{}, fmt.Errorf("resolve OpenRouter credential through authority: %w", err)
	}
	command.Env = map[string]string{"OPENROUTER_API_KEY": credential}
	return e.runner.Run(ctx, command)
}

func resolveOpenRouterCredential(_ context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/openrouter")
	if err != nil {
		return "", err
	}
	return authority.Resolve(identity, "api-key")
}

type mediaImageResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func decodeMediaImage(body []byte) ([]byte, string, error) {
	var response mediaImageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", fmt.Errorf("decode resource-openrouter image response: %w", err)
	}
	if response.Error != nil && strings.TrimSpace(response.Error.Message) != "" {
		return nil, "", fmt.Errorf("resource-openrouter image error: %s", response.Error.Message)
	}
	if len(response.Data) == 0 || strings.TrimSpace(response.Data[0].B64JSON) == "" {
		return nil, "", errors.New("resource-openrouter returned no image output")
	}
	image, err := base64.StdEncoding.DecodeString(response.Data[0].B64JSON)
	if err != nil {
		return nil, "", fmt.Errorf("decode resource-openrouter image bytes: %w", err)
	}
	if len(image) == 0 {
		return nil, "", errors.New("resource-openrouter returned empty image output")
	}
	mediaType := strings.TrimSpace(response.Data[0].MediaType)
	if mediaType == "" {
		mediaType = "image/png"
	}
	return image, mediaType, nil
}

func writeMediaOutput(path string, image []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create media output directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".ai-gateway-media-*")
	if err != nil {
		return fmt.Errorf("create media output temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(image); err != nil {
		cleanup()
		return fmt.Errorf("write media output: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("set media output permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close media output: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit media output: %w", err)
	}
	return nil
}
