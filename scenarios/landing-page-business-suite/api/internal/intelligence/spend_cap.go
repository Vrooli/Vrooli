package intelligence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

var spendUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func validateSpendRequest(limit *SpendLimit, images []AIImage, maxTokens int) error {
	if limit == nil || limit.MaxCostMicros <= 0 || !spendUUID.MatchString(limit.OperationID) || !isLowerHex64(limit.InputFingerprint) || limit.FundingRoute != "vrooli_credits" {
		return errors.New("a valid positive spend ceiling and operation binding are required")
	}
	if maxTokens <= 0 || maxTokens > 4096 || limit.ImageCount != int32(len(images)) || len(images) == 0 || len(images) > 4 || len(limit.ReferenceSHA256) != len(images) {
		return errors.New("spend ceiling image or output bounds are invalid")
	}
	var totalBytes int
	for i, image := range images {
		if image.MediaType != "image/png" && image.MediaType != "image/jpeg" && image.MediaType != "image/webp" {
			return errors.New("unsupported reference image media type")
		}
		decoded, err := base64.StdEncoding.DecodeString(image.Data)
		if err != nil || len(decoded) == 0 || len(decoded) > 12<<20 {
			return errors.New("reference image payload is invalid")
		}
		totalBytes += len(decoded)
		if totalBytes > 10<<20 {
			return errors.New("reference images exceed the bounded payload size")
		}
		sum := sha256.Sum256(decoded)
		got := hex.EncodeToString(sum[:])
		if image.SHA256 != got || limit.ReferenceSHA256[i] != got {
			return errors.New("reference image does not match its immutable digest")
		}
	}
	return nil
}

func (s *MeteredInferenceService) executeSpendCapped(ctx context.Context, userIdentity string, req AIRequest) (*AIResponse, error) {
	if !allowedModels[req.Model] {
		return nil, fmt.Errorf("%w: %s", ErrModelNotAllowed, req.Model)
	}
	if req.SpendLimit.FundingRoute != "vrooli_credits" {
		return nil, errors.New("spend ceiling route is unsupported by LPBS")
	}
	if err := validateSpendRequest(req.SpendLimit, req.Images, req.MaxTokens); err != nil {
		return nil, err
	}
	usage, ok := s.usageService.(CostCapUsageServicer)
	if !ok {
		return nil, errors.New("LPBS current credit authorization is unavailable")
	}
	if s.accountService == nil {
		return nil, errors.New("LPBS account authority is unavailable")
	}
	sub, err := s.accountService.GetSubscriptionContext(ctx, userIdentity)
	if err != nil || sub == nil || sub.PlanTier == nil || strings.TrimSpace(*sub.PlanTier) == "" {
		return nil, errors.New("LPBS current account tier is unavailable")
	}
	client, err := s.getOpenRouterClient(ctx)
	if err != nil {
		return nil, err
	}
	metadataClient, ok := client.(OpenRouterEndpointMetadataClient)
	if !ok {
		return nil, errors.New("current provider endpoint metadata is unavailable")
	}
	endpoints, err := metadataClient.CurrentEndpoints(ctx, req.Model)
	if err != nil {
		return nil, fmt.Errorf("authorize current provider endpoints: %w", err)
	}
	providerPolicy, contextBound, err := capProviderPolicy(endpoints, req.MaxTokens, int(req.SpendLimit.ImageCount), req.SpendLimit.MaxCostMicros)
	if err != nil {
		return nil, err
	}
	controlsJSON, _ := json.Marshal(providerPolicy)
	controlsDigest := sha256.Sum256(controlsJSON)
	reservationID, reserved, multiplier, err := usage.ReserveUSDSpendCap(ctx, userIdentity, *sub.PlanTier, req.SpendLimit.MaxCostMicros, req.SpendLimit.OperationID)
	if err != nil {
		return nil, err
	}
	// Mark the hold indeterminate before dispatch so a process crash cannot let
	// automatic cleanup release a possibly consumed provider budget.
	if err := usage.RetainUncertainReservation(ctx, reservationID); err != nil {
		_ = s.usageService.ReleaseReservation(ctx, reservationID)
		return nil, fmt.Errorf("retain spend-cap reservation before dispatch: %w", err)
	}
	orMessages := make([]OpenRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		orMessages[i] = OpenRouterMessage{Role: msg.Role, Content: msg.Content}
	}
	for i := len(orMessages) - 1; i >= 0; i-- {
		if orMessages[i].Role == "user" {
			parts := []map[string]any{{"type": "text", "text": req.Messages[i].Content}}
			for _, image := range req.Images {
				parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + image.MediaType + ";base64," + image.Data}})
			}
			orMessages[i].Content = parts
			break
		}
	}
	response, err := client.Chat(ctx, OpenRouterChatRequest{Model: req.Model, Messages: orMessages, MaxTokens: req.MaxTokens, Provider: &providerPolicy})
	if err != nil {
		_ = usage.RetainUncertainReservation(ctx, reservationID)
		return nil, fmt.Errorf("provider outcome retained pending spend-cap reconciliation: %w", err)
	}
	refs := append([]string(nil), req.SpendLimit.ReferenceSHA256...)
	authorization := &inferencev1.SpendAuthorization{
		AuthorizationId: req.SpendLimit.OperationID, Status: "authorized", MaxCostMicros: req.SpendLimit.MaxCostMicros,
		InputTokensBound: int32(contextBound - req.MaxTokens), OutputTokensBound: int32(req.MaxTokens), ImageCount: req.SpendLimit.ImageCount,
		ContextSource: "openrouter-current-endpoint-metadata", OperationId: req.SpendLimit.OperationID, InputFingerprint: req.SpendLimit.InputFingerprint,
		ReferenceSha256: refs, FundingRoute: req.SpendLimit.FundingRoute, AccountIdentity: userIdentity, Model: req.Model,
		ProviderControlsSha256: hex.EncodeToString(controlsDigest[:]), ContextWindowTokens: int32(contextBound),
	}
	if !response.Usage.CostReported {
		return &AIResponse{ID: response.ID, Model: response.Model, Content: response.Content, PromptTokens: response.Usage.PromptTokens, CompletionTokens: response.Usage.CompletionTokens, TotalTokens: response.Usage.TotalTokens, FinishReason: response.FinishReason, SpendAuthorization: authorization, AccountingStatus: "pending_reconciliation"}, nil
	}
	if response.Usage.CostMicros < 0 || response.Usage.CostMicros > req.SpendLimit.MaxCostMicros {
		return nil, errors.New("provider outcome retained pending: actual cost exceeds the authorized ceiling")
	}
	actualCredits, err := ceilMulDiv(response.Usage.CostMicros, multiplier, 10000)
	if err != nil || actualCredits > reserved {
		return nil, errors.New("provider outcome retained pending: actual cost exceeds the reserved ceiling")
	}
	if err := finalizeUsageWithCost(s.usageService, ctx, reservationID, actualCredits, req.Metadata.AppBundleKey, req.Model, response.Usage.CostMicros, response.Usage.PromptTokens, response.Usage.CompletionTokens, "openrouter"); err != nil {
		return nil, fmt.Errorf("provider usage retained pending for reconciliation: settle reservation: %w", err)
	}
	actualCost := response.Usage.CostMicros
	return &AIResponse{ID: response.ID, Model: response.Model, Content: response.Content, PromptTokens: response.Usage.PromptTokens, CompletionTokens: response.Usage.CompletionTokens, TotalTokens: response.Usage.TotalTokens, CreditsCharged: actualCredits, FinishReason: response.FinishReason, SpendAuthorization: authorization, AccountingStatus: "settled", ActualCostMicros: &actualCost}, nil
}

func capProviderPolicy(endpoints []OpenRouterEndpointMetadata, output, images int, capMicros int64) (OpenRouterProviderPolicy, int, error) {
	contextBound := int(^uint(0) >> 1)
	eligible := make([]OpenRouterEndpointMetadata, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.ContextTokens <= output || endpoint.ContextTokens <= 0 || int64(endpoint.ContextTokens) > math.MaxInt32 || strings.TrimSpace(endpoint.Provider) == "" {
			continue
		}
		if endpoint.PromptPrice < 0 || endpoint.CompletionPrice < 0 || endpoint.RequestPrice < 0 || endpoint.ImagePrice < 0 ||
			math.IsNaN(endpoint.PromptPrice) || math.IsNaN(endpoint.CompletionPrice) || math.IsNaN(endpoint.RequestPrice) || math.IsNaN(endpoint.ImagePrice) ||
			math.IsInf(endpoint.PromptPrice, 0) || math.IsInf(endpoint.CompletionPrice, 0) || math.IsInf(endpoint.RequestPrice, 0) || math.IsInf(endpoint.ImagePrice, 0) {
			continue
		}
		eligible = append(eligible, endpoint)
		if endpoint.ContextTokens < contextBound {
			contextBound = endpoint.ContextTokens
		}
	}
	if len(eligible) == 0 || contextBound == int(^uint(0)>>1) {
		return OpenRouterProviderPolicy{}, 0, errors.New("current endpoint metadata has no fully priced bounded provider")
	}
	inputBound := contextBound - output
	units := [4]int64{int64(inputBound), int64(output), 1, int64(images)}
	active := [4]bool{true, true, true, true}
	for _, endpoint := range eligible {
		active[2] = active[2] && endpoint.RequestPriceKnown
		active[3] = active[3] && endpoint.ImagePriceKnown
	}
	activeCount := int64(0)
	for _, enabled := range active {
		if enabled {
			activeCount++
		}
	}
	shares := [4]int64{}
	left := capMicros
	for i, enabled := range active {
		if !enabled {
			continue
		}
		shares[i] = capMicros / activeCount
		left -= shares[i]
	}
	for i := len(active) - 1; left > 0; i-- {
		if active[i] {
			shares[i]++
			left--
		}
	}
	prices := [4]float64{}
	for i := range units {
		nano, err := ceilSafeFloorPerUnit(shares[i], units[i])
		if err != nil {
			return OpenRouterProviderPolicy{}, 0, err
		}
		prices[i] = math.Nextafter(float64(nano)/1_000_000_000, 0)
	}
	providers := make([]string, 0, len(eligible))
	for _, endpoint := range eligible {
		if endpoint.PromptPrice <= prices[0] && endpoint.CompletionPrice <= prices[1] && endpoint.RequestPrice <= prices[2] && endpoint.ImagePrice <= prices[3] {
			providers = append(providers, endpoint.Provider)
		}
	}
	sort.Strings(providers)
	providers = uniqueStrings(providers)
	if len(providers) == 0 {
		return OpenRouterProviderPolicy{}, 0, errors.New("current provider prices exceed the requested spend ceiling")
	}
	// Endpoint token rates are USD/token, while OpenRouter's max_price
	// prompt/completion controls are USD per million tokens. Request and image
	// controls remain USD/request and USD/image.
	promptWire := math.Nextafter(prices[0]*1_000_000, 0)
	completionWire := math.Nextafter(prices[1]*1_000_000, 0)
	if math.IsInf(promptWire, 0) || math.IsInf(completionWire, 0) {
		return OpenRouterProviderPolicy{}, 0, errors.New("spend-cap token price overflows provider policy")
	}
	requestWire, imageWire := prices[2], prices[3]
	if !active[2] {
		requestWire = 0
	}
	if !active[3] {
		imageWire = 0
	}
	return OpenRouterProviderPolicy{Only: providers, AllowFallbacks: false, RequireParameters: true, MaxPrice: OpenRouterMaxPrice{Prompt: promptWire, Completion: completionWire, Request: requestWire, Image: imageWire}}, contextBound, nil
}

func ceilSafeFloorPerUnit(micros, units int64) (int64, error) {
	if micros < 0 || units <= 0 {
		return 0, errors.New("invalid spend-cap allocation")
	}
	n := new(big.Int).Mul(big.NewInt(micros), big.NewInt(1000))
	n.Quo(n, big.NewInt(units))
	if !n.IsInt64() {
		return 0, errors.New("spend-cap allocation overflows")
	}
	return n.Int64(), nil
}

func ceilMulDiv(value, multiplier, divisor int64) (int64, error) {
	if value < 0 || multiplier <= 0 || divisor <= 0 {
		return 0, errors.New("invalid credit conversion")
	}
	n := new(big.Int).Mul(big.NewInt(value), big.NewInt(multiplier))
	n.Add(n, big.NewInt(divisor-1))
	n.Quo(n, big.NewInt(divisor))
	if !n.IsInt64() {
		return 0, errors.New("credit conversion overflows")
	}
	return n.Int64(), nil
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, v := range values[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

func isLowerHex64(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
