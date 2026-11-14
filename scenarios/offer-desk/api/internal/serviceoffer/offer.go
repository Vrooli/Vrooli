// Package serviceoffer defines the prototype contract for one service offer
// projected into its human and agent discovery/purchase doors. Offer Desk
// remains the intended owner; callers must treat all public effects as staged
// until the operator approves the exact item.
package serviceoffer

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	pathpkg "path"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

//go:embed testdata
var prototypeFiles embed.FS

// Record is the single source record from which all purchase and discovery
// surfaces are projected. Price and rights are recommendations/gates, not
// authority to charge or publish.
type Record struct {
	ID                  string          `json:"id"`
	Version             int             `json:"version"`
	Name                string          `json:"name"`
	Summary             string          `json:"summary"`
	Audience            string          `json:"audience"`
	IntakeSchema        json.RawMessage `json:"intake_schema"`
	ProposedPrice       Price           `json:"proposed_price"`
	Delivery            Delivery        `json:"delivery"`
	VerificationPromise string          `json:"verification_promise"`
	RefundRule          string          `json:"refund_rule"`
	Sample              Sample          `json:"sample"`
	Legal               LegalGate       `json:"legal"`
	CheckoutMode        string          `json:"checkout_mode"`
	MCPToolName         string          `json:"mcp_tool_name"`
	MPPResourcePath     string          `json:"mpp_resource_path"`
	X402PriceID         string          `json:"x402_price_id"`
	PluginDisposition   string          `json:"plugin_disposition"`
	OperatorApproval    ApprovalState   `json:"operator_approval"`
}

type Price struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type Delivery struct {
	TargetMinutes int    `json:"target_minutes"`
	Format        string `json:"format"`
}

type Sample struct {
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
	Description string `json:"description"`
	NoSignup    bool   `json:"no_signup"`
}

type LegalGate struct {
	ReviewedAt string   `json:"reviewed_at"`
	Status     string   `json:"status"` // open, pass, fail; only an operator may accept rights.
	Notes      []string `json:"notes"`
	Evidence   []string `json:"evidence"`
}

type ApprovalState string

const (
	ApprovalPending  ApprovalState = "pending"
	ApprovalApproved ApprovalState = "approved"
	ApprovalDeclined ApprovalState = "declined"
)

// Page is the staged human/browsing-agent landing surface. It never contains a
// direct transactional URL: a browser agent must use the approved checkout.
type Page struct {
	Slug                  string `json:"slug"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	Audience              string `json:"audience"`
	ProposedPrice         Price  `json:"proposed_price"`
	DeliveryTargetMinutes int    `json:"delivery_target_minutes"`
	VerificationPromise   string `json:"verification_promise"`
	RefundRule            string `json:"refund_rule"`
	FreeSamplePath        string `json:"free_sample_path"`
	CheckoutPath          string `json:"checkout_path"`
	PublicationState      string `json:"publication_state"`
}

// GuestCheckout is a test-mode-only description of a person or browsing
// agent's order. Capture is explicitly deferred until delivery verifies.
type GuestCheckout struct {
	OfferID               string `json:"offer_id"`
	Mode                  string `json:"mode"`
	PaymentMode           string `json:"payment_mode"`
	CaptureWhen           string `json:"capture_when"`
	DeliveryTargetMinutes int    `json:"delivery_target_minutes"`
	RefundRule            string `json:"refund_rule"`
}

type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	PaymentPath string          `json:"payment_path"`
}

// PluginDiscovery is always non-transactional. Platform policy allows digital
// service discovery but does not allow selling that service inside the plugin.
// Any plugin path leads to the service page, never directly to checkout.
type PluginDiscovery struct {
	Disposition  string `json:"disposition"`
	LandingPath  string `json:"landing_path"`
	CheckoutPath string `json:"checkout_path,omitempty"`
}

type MPPResource struct {
	Path        string `json:"path"`
	PriceID     string `json:"price_id"`
	OfferID     string `json:"offer_id"`
	Mode        string `json:"mode"`
	CaptureWhen string `json:"capture_when"`
}

// Surfaces are derived, never independently authored. The prototype supports
// a staged page, sandbox guest checkout, MCP descriptor and test MPP/x402
// declarations from the same record.
type Surfaces struct {
	Page     Page            `json:"page"`
	Checkout GuestCheckout   `json:"checkout"`
	MCP      MCPTool         `json:"mcp"`
	MPP      MPPResource     `json:"mpp"`
	Plugin   PluginDiscovery `json:"plugin"`
	LLMSTxt  string          `json:"llms_txt"`
	OpenAPI  map[string]any  `json:"openapi"`
}

func Validate(record Record) error {
	var problems []string
	if !slugPattern.MatchString(record.ID) {
		problems = append(problems, "id must be a stable lowercase slug")
	}
	if record.Version < 1 {
		problems = append(problems, "version must be positive")
	}
	if strings.TrimSpace(record.Name) == "" || strings.TrimSpace(record.Summary) == "" || strings.TrimSpace(record.Audience) == "" {
		problems = append(problems, "name, summary and audience are required")
	}
	if err := validObjectSchema(record.IntakeSchema); err != nil {
		problems = append(problems, "intake_schema: "+err.Error())
	}
	if record.ProposedPrice.AmountMinor <= 0 || !validCurrency(record.ProposedPrice.Currency) {
		problems = append(problems, "proposed_price requires a positive amount and ISO-style currency")
	}
	if record.Delivery.TargetMinutes <= 0 || strings.TrimSpace(record.Delivery.Format) == "" {
		problems = append(problems, "delivery target and format are required")
	}
	if strings.TrimSpace(record.VerificationPromise) == "" || strings.TrimSpace(record.RefundRule) == "" {
		problems = append(problems, "verification promise and refund rule are required")
	}
	if !validSamplePath(record.Sample.Path) || len(record.Sample.SHA256) != 64 || !record.Sample.NoSignup {
		problems = append(problems, "a hashed, no-signup sample path is required")
	}
	if strings.TrimSpace(record.Sample.Description) == "" {
		problems = append(problems, "sample description is required")
	}
	if record.Legal.Status != "open" && record.Legal.Status != "pass" && record.Legal.Status != "fail" {
		problems = append(problems, "legal status must be open, pass or fail")
	}
	if record.Legal.Status == "fail" {
		problems = append(problems, "failed legal gate declines this offer")
	}
	if record.Legal.Status != "pass" && record.OperatorApproval == ApprovalApproved {
		problems = append(problems, "operator approval cannot clear an unresolved or failed legal gate")
	}
	if record.OperatorApproval != ApprovalPending && record.OperatorApproval != ApprovalApproved && record.OperatorApproval != ApprovalDeclined {
		problems = append(problems, "operator approval must be pending, approved or declined")
	}
	if record.CheckoutMode != "test" {
		problems = append(problems, "checkout_mode must be test until a separate exact live approval")
	}
	if !slugPattern.MatchString(record.MCPToolName) {
		problems = append(problems, "mcp_tool_name must be a stable lowercase slug")
	}
	if !strings.HasPrefix(record.MPPResourcePath, "/") || strings.Contains(record.MPPResourcePath, "..") {
		problems = append(problems, "mpp_resource_path must be a local absolute route")
	}
	if !slugPattern.MatchString(record.X402PriceID) {
		problems = append(problems, "x402_price_id must be a stable lowercase slug")
	}
	if record.PluginDisposition != "free-discovery-only" && record.PluginDisposition != "physical-commerce" && record.PluginDisposition != "none" {
		problems = append(problems, "plugin disposition must respect platform commerce policy")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func Project(record Record, sampleFiles fs.FS) (Surfaces, error) {
	if err := Validate(record); err != nil {
		return Surfaces{}, err
	}
	if err := VerifySample(sampleFiles, record.Sample); err != nil {
		return Surfaces{}, fmt.Errorf("verify public sample: %w", err)
	}
	if record.OperatorApproval == ApprovalDeclined {
		return Surfaces{}, errors.New("declined offer cannot be projected")
	}
	pagePath := "/services/" + record.ID
	checkoutPath := "/checkout/" + record.ID
	page := Page{
		Slug: record.ID, Title: record.Name, Description: record.Summary, Audience: record.Audience,
		ProposedPrice: record.ProposedPrice, DeliveryTargetMinutes: record.Delivery.TargetMinutes,
		VerificationPromise: record.VerificationPromise, RefundRule: record.RefundRule,
		FreeSamplePath: "/samples/" + record.Sample.Path, CheckoutPath: checkoutPath, PublicationState: "staged",
	}
	checkout := GuestCheckout{
		OfferID: record.ID, Mode: "test", PaymentMode: "authorize-then-capture",
		CaptureWhen: "after-delivery-verification", DeliveryTargetMinutes: record.Delivery.TargetMinutes,
		RefundRule: record.RefundRule,
	}
	mcp := MCPTool{Name: record.MCPToolName, Description: record.Summary, InputSchema: cloneJSON(record.IntakeSchema), PaymentPath: "checkout:" + record.ID}
	mpp := MPPResource{Path: record.MPPResourcePath, PriceID: record.X402PriceID, OfferID: record.ID, Mode: "test", CaptureWhen: "after-delivery-verification"}
	openapi := map[string]any{
		"openapi": "3.1.0", "info": map[string]any{"title": record.Name, "version": fmt.Sprintf("%d", record.Version), "description": record.Summary},
		"paths": map[string]any{record.MPPResourcePath: map[string]any{"post": map[string]any{
			"operationId": record.MCPToolName, "summary": record.Summary,
			"x-vrooli-offer-id": record.ID, "x-vrooli-price-id": record.X402PriceID,
			"x-vrooli-payment-mode": "test", "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": record.IntakeSchema}}},
		}}, checkoutPath: map[string]any{"post": map[string]any{"operationId": "checkout_" + record.ID, "x-vrooli-offer-id": record.ID, "x-vrooli-payment-mode": "test"}}},
	}
	llms := fmt.Sprintf("# Services\n\n- [%s](%s): %s\n  - Sample: /samples/%s\n  - Tool: %s\n", record.Name, pagePath, record.Summary, record.Sample.Path, record.MCPToolName)
	plugin := PluginDiscovery{Disposition: record.PluginDisposition, LandingPath: pagePath}
	return Surfaces{Page: page, Checkout: checkout, MCP: mcp, MPP: mpp, Plugin: plugin, LLMSTxt: llms, OpenAPI: openapi}, nil
}

func validObjectSchema(raw json.RawMessage) error {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return errors.New("must be valid JSON")
	}
	if value["type"] != "object" {
		return errors.New("root type must be object")
	}
	if _, ok := value["properties"].(map[string]any); !ok {
		return errors.New("properties must be an object")
	}
	return nil
}

func validCurrency(currency string) bool {
	return len(currency) == 3 && currency == strings.ToUpper(currency)
}

func validSamplePath(path string) bool {
	if strings.TrimSpace(path) == "" || path[0] == '/' {
		return false
	}
	clean := pathpkg.Clean(path)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func cloneJSON(raw json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), raw...)
}

// CheckoutURL returns the relative guest checkout path and rejects caller
// supplied origin URLs, so an offer record cannot smuggle an unapproved vendor
// or direct payment URL into a page projection.
func CheckoutURL(path string) error {
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/checkout/") {
		return errors.New("checkout path must be a relative same-origin path")
	}
	return nil
}

// VerifySample checks that a declared public sample exists beneath root and
// matches its content digest. It refuses absolute paths and traversal so offer
// records cannot read arbitrary local files when rendered.
func VerifySample(sampleFiles fs.FS, sample Sample) error {
	if !validSamplePath(sample.Path) || !fs.ValidPath(sample.Path) {
		return errors.New("sample path must be relative and stay within its root")
	}
	data, err := fs.ReadFile(sampleFiles, sample.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("sample file does not exist")
		}
		return fmt.Errorf("read sample file: %w", err)
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if !strings.EqualFold(actual, sample.SHA256) {
		return fmt.Errorf("sample digest mismatch: expected %s, got %s", sample.SHA256, actual)
	}
	return nil
}

// Prototype returns the checked-in report record and its embedded public
// sample. It exists only to rehearse the proposed offer contract locally; it
// is not a live offer or a publication authorization.
func Prototype() (Record, fs.FS, error) {
	var record Record
	data, err := prototypeFiles.ReadFile("testdata/company-market-scan-offer.json")
	if err != nil {
		return Record{}, nil, fmt.Errorf("read prototype offer: %w", err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, nil, fmt.Errorf("decode prototype offer: %w", err)
	}
	sampleFiles, err := fs.Sub(prototypeFiles, "testdata")
	if err != nil {
		return Record{}, nil, fmt.Errorf("open prototype sample: %w", err)
	}
	return record, sampleFiles, nil
}
