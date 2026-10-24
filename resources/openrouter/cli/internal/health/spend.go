package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

// SpendBinding is the immutable operation identity supplied by an authorized
// caller. It is recorded locally and never sent to the model provider.
type SpendBinding struct {
	MaxCostMicros    int64
	OperationID      string
	InputFingerprint string
	ReferenceSHA256  []string
	FundingRoute     string
	AccountIdentity  string
}

// SpendAuthorization is owner-authored evidence that the provider request was
// constrained by a user maximum. It intentionally contains no price estimate.
type SpendAuthorization struct {
	AuthorizationID        string    `json:"authorization_id"`
	Status                 string    `json:"status"`
	MaxCostMicros          int64     `json:"max_cost_micros"`
	InputTokensBound       int       `json:"input_tokens_bound"`
	OutputTokensBound      int       `json:"output_tokens_bound"`
	ImageCount             int       `json:"image_count"`
	ContextSource          string    `json:"context_source"`
	AuthorizedAt           time.Time `json:"authorized_at"`
	OperationID            string    `json:"operation_id"`
	InputFingerprint       string    `json:"input_fingerprint"`
	ReferenceSHA256        []string  `json:"reference_sha256"`
	FundingRoute           string    `json:"funding_route"`
	AccountIdentity        string    `json:"account_identity"`
	Model                  string    `json:"model"`
	ProviderControlsSHA256 string    `json:"provider_controls_sha256"`
	ContextWindowTokens    int       `json:"context_window_tokens"`
}

type MaxPrice struct {
	Prompt     json.Number `json:"prompt"`
	Completion json.Number `json:"completion"`
	Request    json.Number `json:"request"`
	Image      json.Number `json:"image"`
}

type ProviderSpendPolicy struct {
	AllowFallbacks    bool     `json:"allow_fallbacks"`
	RequireParameters bool     `json:"require_parameters"`
	Only              []string `json:"only"`
	MaxPrice          MaxPrice `json:"max_price"`
}

type spendEndpoint struct {
	ProviderName        string                     `json:"provider_name"`
	ContextLength       int                        `json:"context_length"`
	MaxCompletionTokens int                        `json:"max_completion_tokens"`
	Pricing             map[string]json.RawMessage `json:"pricing"`
}

// AuthorizeSpendCap resolves current endpoint context/pricing metadata and
// creates a hard-cap provider policy. Static model catalogs are not an
// authorization source. Any missing or unknown price class fails closed.
func AuthorizeSpendCap(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model string, allowedEndpoints []string, outputTokens, imageCount int, binding SpendBinding) (ProviderSpendPolicy, SpendAuthorization, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if !creds.Valid() || binding.MaxCostMicros <= 0 || outputTokens <= 0 || imageCount < 1 || imageCount > 4 || binding.FundingRoute != "byok" || binding.AccountIdentity != "vrooli/openrouter" || !validOperationID(binding.OperationID) || !validDigest(binding.InputFingerprint) || len(binding.ReferenceSHA256) != imageCount {
		return ProviderSpendPolicy{}, SpendAuthorization{}, errors.New("spend cap binding is incomplete")
	}
	for _, digest := range binding.ReferenceSHA256 {
		if !validDigest(digest) {
			return ProviderSpendPolicy{}, SpendAuthorization{}, errors.New("spend reference digest is invalid")
		}
	}
	if len(allowedEndpoints) == 0 {
		return ProviderSpendPolicy{}, SpendAuthorization{}, errors.New("spend cap requires a pinned endpoint allow-list")
	}
	modelParts := strings.Split(strings.TrimSpace(model), "/")
	if len(modelParts) != 2 || modelParts[0] == "" || modelParts[1] == "" {
		return ProviderSpendPolicy{}, SpendAuthorization{}, errors.New("spend cap requires a canonical author/model slug")
	}
	endpoint := strings.TrimRight(runtime.APIBaseURL, "/") + "/models/" + url.PathEscape(modelParts[0]) + "/" + url.PathEscape(modelParts[1]) + "/endpoints"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ProviderSpendPolicy{}, SpendAuthorization{}, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return ProviderSpendPolicy{}, SpendAuthorization{}, fmt.Errorf("resolve current OpenRouter endpoint metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ProviderSpendPolicy{}, SpendAuthorization{}, fmt.Errorf("current OpenRouter endpoint metadata unavailable: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data []spendEndpoint `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ProviderSpendPolicy{}, SpendAuthorization{}, fmt.Errorf("decode current endpoint metadata: %w", err)
	}
	allowed := make(map[string]bool, len(allowedEndpoints))
	for _, name := range allowedEndpoints {
		if name = strings.TrimSpace(name); name != "" {
			allowed[strings.ToLower(name)] = true
		}
	}
	var providers []string
	contextWindow := 0
	for _, candidate := range payload.Data {
		provider := strings.TrimSpace(candidate.ProviderName)
		if provider == "" || !allowed[strings.ToLower(provider)] || candidate.ContextLength <= outputTokens || candidate.MaxCompletionTokens > 0 && candidate.MaxCompletionTokens < outputTokens || !completePriceClasses(candidate.Pricing) {
			continue
		}
		providers = append(providers, provider)
		if contextWindow == 0 || candidate.ContextLength < contextWindow {
			contextWindow = candidate.ContextLength
		}
	}
	providers = uniqueSorted(providers)
	inputBound := contextWindow - outputTokens
	if len(providers) == 0 || inputBound <= 0 {
		return ProviderSpendPolicy{}, SpendAuthorization{}, errors.New("no current allowed endpoint has bounded context and all supported price classes")
	}
	limits, err := AllocateMaxPrice(binding.MaxCostMicros, inputBound, outputTokens, imageCount)
	if err != nil {
		return ProviderSpendPolicy{}, SpendAuthorization{}, err
	}
	stamp := time.Now().UTC()
	controls, _ := json.Marshal(struct {
		Policy ProviderSpendPolicy `json:"policy"`
		Model  string              `json:"model"`
	}{ProviderSpendPolicy{AllowFallbacks: false, RequireParameters: true, Only: providers, MaxPrice: limits}, model})
	controlsDigest := sha256.Sum256(controls)
	identity, _ := json.Marshal(struct {
		Binding   SpendBinding `json:"binding"`
		Providers []string     `json:"providers"`
		Context   int          `json:"context"`
		Input     int          `json:"input"`
		Output    int          `json:"output"`
		Controls  string       `json:"controls"`
		At        string       `json:"at"`
	}{binding, providers, contextWindow, inputBound, outputTokens, hex.EncodeToString(controlsDigest[:]), stamp.Format(time.RFC3339Nano)})
	digest := sha256.Sum256(identity)
	return ProviderSpendPolicy{AllowFallbacks: false, RequireParameters: true, Only: providers, MaxPrice: limits}, SpendAuthorization{AuthorizationID: hex.EncodeToString(digest[:]), Status: "authorized", MaxCostMicros: binding.MaxCostMicros, InputTokensBound: inputBound, OutputTokensBound: outputTokens, ImageCount: imageCount, ContextSource: "openrouter-current-endpoint-metadata", AuthorizedAt: stamp, OperationID: binding.OperationID, InputFingerprint: binding.InputFingerprint, ReferenceSHA256: append([]string(nil), binding.ReferenceSHA256...), FundingRoute: binding.FundingRoute, AccountIdentity: binding.AccountIdentity, Model: model, ProviderControlsSHA256: hex.EncodeToString(controlsDigest[:]), ContextWindowTokens: contextWindow}, nil
}

// AllocateMaxPrice deterministically divides the USD ceiling across prompt,
// completion, request, and image price classes using integer micro-dollars.
// The provider's aggregate worst case is therefore no greater than the cap.
func AllocateMaxPrice(maxCostMicros int64, inputTokens, outputTokens, imageCount int) (MaxPrice, error) {
	if maxCostMicros <= 0 || inputTokens <= 0 || outputTokens <= 0 || imageCount <= 0 {
		return MaxPrice{}, errors.New("spend cap and token/image bounds must be positive")
	}
	share := maxCostMicros / 4
	if share <= 0 {
		return MaxPrice{}, errors.New("spend ceiling is too small to bound all four provider price classes")
	}
	prompt, err := ratioJSONNumber(share, int64(inputTokens), 12)
	if err != nil {
		return MaxPrice{}, err
	}
	completion, err := ratioJSONNumber(share, int64(outputTokens), 12)
	if err != nil {
		return MaxPrice{}, err
	}
	request, err := microsUSDJSONNumber(share)
	if err != nil {
		return MaxPrice{}, err
	}
	image, err := microsUSDJSONNumber(share / int64(imageCount))
	if err != nil {
		return MaxPrice{}, err
	}
	return MaxPrice{Prompt: prompt, Completion: completion, Request: request, Image: image}, nil
}

func ratioJSONNumber(budgetMicros, units int64, places int) (json.Number, error) {
	if budgetMicros < 0 || units <= 0 || places < 0 || places > 18 {
		return "", errors.New("invalid spend ratio")
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	numerator := new(big.Int).Mul(big.NewInt(budgetMicros), scale)
	quotient := new(big.Int).Quo(numerator, big.NewInt(units)) // floor protects the ceiling
	whole, fraction := new(big.Int), new(big.Int)
	whole.QuoRem(quotient, scale, fraction)
	if places == 0 || fraction.Sign() == 0 {
		return json.Number(whole.String()), nil
	}
	fracText := fraction.String()
	if len(fracText) < places {
		fracText = strings.Repeat("0", places-len(fracText)) + fracText
	}
	frac := strings.TrimRight(fracText, "0")
	return json.Number(whole.String() + "." + frac), nil
}

func microsUSDJSONNumber(micros int64) (json.Number, error) {
	if micros < 0 {
		return "", errors.New("negative spend class cap")
	}
	return json.Number(fmt.Sprintf("%d.%06d", micros/1_000_000, micros%1_000_000)), nil
}

func completePriceClasses(pricing map[string]json.RawMessage) bool {
	for key, raw := range pricing {
		switch key {
		case "prompt", "completion", "request", "image":
		default:
			if len(raw) > 0 && string(raw) != `""` {
				return false
			}
		}
	}
	for _, key := range []string{"prompt", "completion", "request", "image"} {
		raw, ok := pricing[key]
		if !ok {
			return false
		}
		var number json.Number
		if err := json.Unmarshal(raw, &number); err != nil {
			var text string
			if json.Unmarshal(raw, &text) != nil {
				return false
			}
			number = json.Number(text)
		}
		rat, ok := new(big.Rat).SetString(number.String())
		if !ok || rat.Sign() < 0 {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func validOperationID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if !seen[strings.ToLower(v)] {
			seen[strings.ToLower(v)] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
