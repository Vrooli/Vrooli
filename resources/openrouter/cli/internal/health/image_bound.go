package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

// ImageControls are the per-request levers that decide how much money OpenRouter
// reserves before it dispatches an image request. The images endpoint has no
// max_price or max_tokens field (unknown fields are dropped), so the only
// enforceable bound is the completion-token ceiling on chat completions:
// without one OpenRouter reserves max_completion_tokens * image_output price,
// which for token-priced image models is several dollars even though a real
// 1K/2K image uses about a thousand tokens.
type ImageControls struct {
	// Resolution and Size come from the role's request_defaults.
	Resolution string
	Size       string
	// MaxOutputTokens is the role's policy-declared completion-token ceiling,
	// sized to the model's real per-image output. Zero means the role declares none.
	MaxOutputTokens int
	// MaxCostMicros is the caller's USD ceiling for this one request in
	// micro-dollars. Zero means the caller supplied no ceiling.
	MaxCostMicros int64
}

// ImageOutputPricing is the current image-output token price of a model and the
// transport that can bound it, resolved from OpenRouter's endpoint metadata
// (never from a static catalog).
type ImageOutputPricing struct {
	// PricePerToken is the highest image_output USD/token among the model's endpoints,
	// so a bound derived from it holds whichever endpoint serves the request.
	PricePerToken *big.Rat
	// EndpointMaxTokens is the largest max_completion_tokens advertised by the
	// endpoints; it is the amount OpenRouter reserves when no bound is sent.
	EndpointMaxTokens int
	// ChatBoundable reports that the model returns images through chat completions
	// and every token-priced endpoint accepts max_tokens there, so a token bound
	// is enforceable.
	ChatBoundable bool
}

// ImageReservation describes the completion-token bound sent to OpenRouter and
// what it implies, so a refusal can state the amount that was actually requested.
type ImageReservation struct {
	MaxTokens       int
	ReservedMicros  int64
	UnboundedMicros int64
}

// BoundImageOutputTokens derives the completion-token ceiling for an image
// request: the role's policy default, lowered to what the caller's USD ceiling
// buys at the model's image-output price, and never above what the endpoint can
// produce. A zero result with a nil error means nothing bounds the request.
func BoundImageOutputTokens(controls ImageControls, pricing ImageOutputPricing) (ImageReservation, error) {
	tokens := controls.MaxOutputTokens
	if controls.MaxCostMicros > 0 {
		if pricing.PricePerToken == nil || pricing.PricePerToken.Sign() <= 0 {
			return ImageReservation{}, errors.New("spend ceiling cannot bound this image model: no image-output token price is published")
		}
		budget := new(big.Rat).SetFrac64(controls.MaxCostMicros, 1_000_000)
		quotient := new(big.Rat).Quo(budget, pricing.PricePerToken)
		capTokens := new(big.Int).Quo(quotient.Num(), quotient.Denom()) // floor protects the ceiling
		if capTokens.Sign() <= 0 || !capTokens.IsInt64() {
			return ImageReservation{}, fmt.Errorf("spend ceiling of %s USD is below the price of one output token", microsString(controls.MaxCostMicros))
		}
		if tokens == 0 || capTokens.Int64() < int64(tokens) {
			tokens = int(capTokens.Int64())
		}
	}
	if pricing.EndpointMaxTokens > 0 && tokens > pricing.EndpointMaxTokens {
		tokens = pricing.EndpointMaxTokens
	}
	reservation := ImageReservation{MaxTokens: tokens}
	if pricing.PricePerToken != nil && pricing.PricePerToken.Sign() > 0 {
		reservation.ReservedMicros = ceilMicros(new(big.Rat).Mul(pricing.PricePerToken, new(big.Rat).SetInt64(int64(tokens))))
		reservation.UnboundedMicros = ceilMicros(new(big.Rat).Mul(pricing.PricePerToken, new(big.Rat).SetInt64(int64(pricing.EndpointMaxTokens))))
	}
	return reservation, nil
}

func ceilMicros(usd *big.Rat) int64 {
	micros := new(big.Rat).Mul(usd, big.NewRat(1_000_000, 1))
	whole := new(big.Int).Quo(micros.Num(), micros.Denom())
	if new(big.Rat).SetInt(whole).Cmp(micros) < 0 {
		whole.Add(whole, big.NewInt(1))
	}
	if !whole.IsInt64() {
		return 0
	}
	return whole.Int64()
}

func microsString(micros int64) string {
	return fmt.Sprintf("%d.%06d", micros/1_000_000, micros%1_000_000)
}

// ResolveImageOutputPricing reads the model's current endpoint metadata. A model
// with no token-priced image output returns nil pricing and no error: nothing
// token-based can be bounded for it.
func ResolveImageOutputPricing(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model string) (ImageOutputPricing, error) {
	if client == nil {
		client = http.DefaultClient
	}
	parts := strings.Split(strings.TrimSpace(model), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ImageOutputPricing{}, errors.New("image model must be a canonical author/model slug")
	}
	endpoint := strings.TrimRight(runtime.APIBaseURL, "/") + "/models/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/endpoints"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ImageOutputPricing{}, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return ImageOutputPricing{}, fmt.Errorf("resolve current OpenRouter endpoint metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ImageOutputPricing{}, fmt.Errorf("current OpenRouter endpoint metadata unavailable: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Architecture struct {
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Endpoints []struct {
				spendEndpoint
				SupportedParameters []string `json:"supported_parameters"`
			} `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ImageOutputPricing{}, fmt.Errorf("decode current endpoint metadata: %w", err)
	}
	var pricing ImageOutputPricing
	allAcceptMaxTokens := true
	for _, candidate := range payload.Data.Endpoints {
		price, ok := parsePrice(candidate.Pricing["image_output"])
		if !ok || price.Sign() <= 0 {
			continue
		}
		if pricing.PricePerToken == nil || price.Cmp(pricing.PricePerToken) > 0 {
			pricing.PricePerToken = price
		}
		if candidate.MaxCompletionTokens > pricing.EndpointMaxTokens {
			pricing.EndpointMaxTokens = candidate.MaxCompletionTokens
		}
		if !slices.Contains(candidate.SupportedParameters, "max_tokens") {
			allAcceptMaxTokens = false
		}
	}
	outputs := payload.Data.Architecture.OutputModalities
	pricing.ChatBoundable = pricing.PricePerToken != nil && allAcceptMaxTokens && slices.Contains(outputs, "image") && slices.Contains(outputs, "text")
	return pricing, nil
}

func parsePrice(raw json.RawMessage) (*big.Rat, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return nil, false
		}
		number = json.Number(text)
	}
	rat, ok := new(big.Rat).SetString(number.String())
	return rat, ok
}

// describeImageRefusal renders the provider's own refusal together with what
// Vrooli asked OpenRouter to reserve, so "the reservation is too large" can be
// told apart from "the key or account limit is exhausted". The provider's
// error.metadata.limit_source names the exhausted limit when OpenRouter sends it.
func describeImageRefusal(body []byte, reservation *ImageReservation) string {
	var failure struct {
		Error struct {
			Message  string         `json:"message"`
			Metadata map[string]any `json:"metadata"`
		} `json:"error"`
	}
	detail := ""
	var limitSource, reason string
	if json.Unmarshal(body, &failure) == nil {
		detail = strings.TrimSpace(failure.Error.Message)
		limitSource, _ = failure.Error.Metadata["limit_source"].(string)
		reason, _ = failure.Error.Metadata["reason"].(string)
	}
	var notes []string
	if limitSource != "" {
		notes = append(notes, "limit_source="+limitSource)
	}
	if reason != "" {
		notes = append(notes, "reason="+reason)
	}
	switch {
	case reservation == nil:
		notes = append(notes, "no output-token bound was sent")
	case reservation.MaxTokens == 0:
		notes = append(notes, fmt.Sprintf("no output-token bound was sent; the endpoint reserves its full completion allowance (up to $%s)", microsString(reservation.UnboundedMicros)))
	default:
		notes = append(notes, fmt.Sprintf("requested reservation: max_tokens=%d, at most $%s (unbounded worst case $%s)", reservation.MaxTokens, microsString(reservation.ReservedMicros), microsString(reservation.UnboundedMicros)))
	}
	return strings.TrimSpace(detail + " [" + strings.Join(notes, "; ") + "]")
}

// imagePlan is the resolved transport and reservation for one image request.
type imagePlan struct {
	viaChat     bool
	reservation *ImageReservation
}

// planImageRequest decides how the request is bounded. With no policy bound and
// no caller ceiling nothing is resolved and the images endpoint is used as-is.
// A bound is sent only through chat completions; when the model cannot take one,
// a caller ceiling is honest only if the endpoint's own worst case already fits.
func planImageRequest(ctx context.Context, client HTTPClient, runtime resourceenv.Runtime, creds auth.Credentials, model string, outputCount int, controls ImageControls) (imagePlan, error) {
	if controls.MaxOutputTokens <= 0 && controls.MaxCostMicros <= 0 {
		return imagePlan{}, nil
	}
	if outputCount > 1 {
		controls.MaxOutputTokens *= outputCount
	}
	pricing, err := ResolveImageOutputPricing(ctx, client, runtime, creds, model)
	if err != nil {
		if controls.MaxCostMicros > 0 {
			// A caller ceiling is only honest when it can be priced; fail closed.
			return imagePlan{}, fmt.Errorf("cannot bound image request to the spend ceiling: %w", err)
		}
		return imagePlan{}, nil
	}
	reservation, err := BoundImageOutputTokens(controls, pricing)
	if err != nil {
		return imagePlan{}, err
	}
	// Chat completions return one image per request, so a multi-image request stays
	// on the images endpoint.
	if pricing.ChatBoundable && reservation.MaxTokens > 0 && outputCount <= 1 {
		return imagePlan{viaChat: true, reservation: &reservation}, nil
	}
	// No bound can be sent: nothing reduces the endpoint's reservation.
	reservation.MaxTokens = 0
	reservation.ReservedMicros = reservation.UnboundedMicros
	if controls.MaxCostMicros > 0 && (reservation.UnboundedMicros == 0 || reservation.UnboundedMicros > controls.MaxCostMicros) {
		return imagePlan{}, fmt.Errorf("spend ceiling of %s USD cannot be enforced for %s: its provider reserves up to %s USD and accepts no output bound", microsString(controls.MaxCostMicros), model, microsString(reservation.UnboundedMicros))
	}
	return imagePlan{reservation: &reservation}, nil
}

// chatImagePayload is the chat-completions form of an image request. max_tokens
// is what makes the provider's reservation bounded.
func chatImagePayload(model, prompt, inputDataURL string, controls ImageControls, maxTokens int) map[string]any {
	var content any = prompt
	if inputDataURL != "" {
		content = []any{
			map[string]any{"type": "text", "text": prompt},
			map[string]any{"type": "image_url", "image_url": map[string]string{"url": inputDataURL}},
		}
	}
	payload := map[string]any{
		"model":      model,
		"messages":   []any{map[string]any{"role": "user", "content": content}},
		"modalities": []string{"image", "text"},
		"max_tokens": maxTokens,
	}
	if resolution := strings.TrimSpace(controls.Resolution); resolution != "" {
		payload["image_config"] = map[string]string{"image_size": resolution}
	}
	return payload
}

// chatImageResponseAsImages rewrites a chat-completions image response into the
// images-endpoint shape so every downstream consumer (raw and gateway-v1) is
// transport-agnostic. Usage is carried over verbatim so cost evidence survives
// even when no image came back.
func chatImageResponseAsImages(body []byte, reservation *ImageReservation) ([]byte, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Images []struct {
					ImageURL struct {
						URL string `json:"url"`
					} `json:"image_url"`
				} `json:"images"`
			} `json:"message"`
			Error *chatBodyError `json:"error"`
		} `json:"choices"`
		Error *chatBodyError  `json:"error"`
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode OpenRouter chat image response: %w", err)
	}
	type outputImage struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type,omitempty"`
	}
	images := []outputImage{}
	for _, choice := range response.Choices {
		for _, image := range choice.Message.Images {
			mediaType, data, ok := splitDataURL(image.ImageURL.URL)
			if !ok {
				continue
			}
			images = append(images, outputImage{B64JSON: data, MediaType: mediaType})
		}
	}
	if len(images) == 0 {
		// A refusal reported inside a 200 body is still a provider failure.
		failure := response.Error
		if failure == nil && len(response.Choices) > 0 {
			failure = response.Choices[0].Error
		}
		if failure != nil && strings.TrimSpace(failure.Message) != "" {
			status := failure.Code
			if status < 400 || status > 599 {
				status = http.StatusBadGateway
			}
			return nil, newHTTPProviderError(status, "", describeImageRefusal(body, reservation))
		}
	}
	out := struct {
		Data  []outputImage   `json:"data"`
		Usage json.RawMessage `json:"usage,omitempty"`
	}{Data: images, Usage: response.Usage}
	return json.Marshal(out)
}

type chatBodyError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func splitDataURL(raw string) (mediaType, data string, ok bool) {
	rest, found := strings.CutPrefix(strings.TrimSpace(raw), "data:")
	if !found {
		return "", "", false
	}
	header, data, found := strings.Cut(rest, ",")
	if !found || data == "" {
		return "", "", false
	}
	mediaType, encoding, _ := strings.Cut(header, ";")
	if encoding != "base64" {
		return "", "", false
	}
	return mediaType, data, true
}
