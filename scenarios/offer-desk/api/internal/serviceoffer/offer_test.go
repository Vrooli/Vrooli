package serviceoffer

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestOneRecordProjectsConsistentStagedAgentAndHumanDoors(t *testing.T) {
	record, samples, err := Prototype()
	require.NoError(t, err)
	surfaces, err := Project(record, samples)
	require.NoError(t, err)

	require.Equal(t, record.ID, surfaces.Page.Slug)
	require.JSONEq(t, string(record.IntakeSchema), string(surfaces.Page.InputSchema))
	require.Equal(t, record.Limits, surfaces.Page.Limits)
	require.Equal(t, record.PrivacyTerms, surfaces.Page.PrivacyTerms)
	require.Equal(t, record.Legal.Status, surfaces.Page.LegalStatus)
	require.Equal(t, record.OperatorApproval, surfaces.Page.OperatorApproval)
	require.Equal(t, record.ProposedPrice, surfaces.Page.ProposedPrice)
	require.Equal(t, record.Delivery.TargetMinutes, surfaces.Page.DeliveryTargetMinutes)
	require.Equal(t, record.VerificationPromise, surfaces.Page.VerificationPromise)
	require.Equal(t, record.RefundRule, surfaces.Checkout.RefundRule)
	require.Equal(t, "test", surfaces.Checkout.Mode)
	require.Equal(t, "authorize-then-capture", surfaces.Checkout.PaymentMode)
	require.Equal(t, "after-delivery-verification", surfaces.Checkout.CaptureWhen)
	require.Equal(t, record.MCPToolName, surfaces.MCP.Name)
	require.Equal(t, record.ID, surfaces.MPP.OfferID)
	require.Equal(t, record.X402PriceID, surfaces.MPP.PriceID)
	require.Equal(t, "test", surfaces.MPP.Mode)
	require.Equal(t, "free-discovery-only", surfaces.Plugin.Disposition)
	require.Empty(t, surfaces.Plugin.CheckoutPath, "plugin discovery must not expose a direct digital-service checkout")
	require.Equal(t, surfaces.Page.Slug, surfaces.MPP.OfferID)
	require.Contains(t, surfaces.LLMSTxt, surfaces.Page.Slug)
	require.Contains(t, surfaces.LLMSTxt, surfaces.MCP.Name)

	body, err := json.Marshal(surfaces.OpenAPI)
	require.NoError(t, err)
	require.Contains(t, string(body), record.ID)
	require.Contains(t, string(body), record.X402PriceID)
	require.Contains(t, string(body), `"x-vrooli-payment-mode":"test"`)
	require.Equal(t, "/samples/company-market-scan.md", surfaces.Page.FreeSamplePath)
}

func TestProjectRejectsFailedRightsGateAndLiveCheckout(t *testing.T) {
	record, samples, err := Prototype()
	require.NoError(t, err)
	record.Legal.Status = "fail"
	_, err = Project(record, samples)
	require.ErrorContains(t, err, "failed legal gate declines")

	record, samples, err = Prototype()
	require.NoError(t, err)
	record.Legal.Status = "pass"
	record.OperatorApproval = ApprovalApproved
	record.CheckoutMode = "live"
	_, err = Project(record, samples)
	require.ErrorContains(t, err, "checkout_mode must be test")
}

func TestProjectRefusesMissingOrChangedSample(t *testing.T) {
	record, _, err := Prototype()
	require.NoError(t, err)
	_, err = Project(record, fstest.MapFS{})
	require.ErrorContains(t, err, "sample file does not exist")

	record.Sample.SHA256 = strings.Repeat("0", 64)
	changed := fstest.MapFS{"company-market-scan.md": &fstest.MapFile{Data: []byte("changed sample")}}
	_, err = Project(record, changed)
	require.ErrorContains(t, err, "sample digest mismatch")
}

func TestValidateRequiresCompleteOfferContractAndHonestGates(t *testing.T) {
	caseList := []struct {
		name string
		edit func(*Record)
		want string
	}{
		{name: "stable id", edit: func(record *Record) { record.ID = "bad id" }, want: "stable lowercase slug"},
		{name: "positive version", edit: func(record *Record) { record.Version = 0 }, want: "version must be positive"},
		{name: "descriptive fields", edit: func(record *Record) { record.Name = " " }, want: "name, summary and audience are required"},
		{name: "valid input schema", edit: func(record *Record) { record.IntakeSchema = json.RawMessage("nope") }, want: "must be valid JSON"},
		{name: "object input schema", edit: func(record *Record) { record.IntakeSchema = json.RawMessage(`{"type":"string","properties":{}}`) }, want: "root type must be object"},
		{name: "schema properties", edit: func(record *Record) { record.IntakeSchema = json.RawMessage(`{"type":"object"}`) }, want: "properties must be an object"},
		{name: "price", edit: func(record *Record) { record.ProposedPrice.AmountMinor = 0 }, want: "positive amount"},
		{name: "currency", edit: func(record *Record) { record.ProposedPrice.Currency = "usd" }, want: "ISO-style currency"},
		{name: "delivery", edit: func(record *Record) { record.Delivery.TargetMinutes = 0 }, want: "delivery target and format"},
		{name: "verification and refund", edit: func(record *Record) { record.VerificationPromise = "" }, want: "verification promise and refund rule"},
		{name: "sample path", edit: func(record *Record) { record.Sample.Path = "../private.md" }, want: "hashed, no-signup sample"},
		{name: "sample digest", edit: func(record *Record) { record.Sample.SHA256 = "short" }, want: "hashed, no-signup sample"},
		{name: "sample signup", edit: func(record *Record) { record.Sample.NoSignup = false }, want: "hashed, no-signup sample"},
		{name: "sample description", edit: func(record *Record) { record.Sample.Description = " " }, want: "sample description is required"},
		{name: "legal enum", edit: func(record *Record) { record.Legal.Status = "unknown" }, want: "legal status must be"},
		{name: "approval cannot bypass legal gate", edit: func(record *Record) { record.OperatorApproval = ApprovalApproved }, want: "cannot clear an unresolved"},
		{name: "approval enum", edit: func(record *Record) { record.OperatorApproval = "maybe" }, want: "operator approval must be"},
		{name: "checkout remains sandboxed", edit: func(record *Record) { record.CheckoutMode = "live" }, want: "checkout_mode must be test"},
		{name: "stable MCP tool", edit: func(record *Record) { record.MCPToolName = "bad tool" }, want: "mcp_tool_name must be"},
		{name: "local MPP path", edit: func(record *Record) { record.MPPResourcePath = "https://pay.example" }, want: "mpp_resource_path must be"},
		{name: "stable x402 price id", edit: func(record *Record) { record.X402PriceID = "bad id" }, want: "x402_price_id must be"},
		{name: "plugin policy", edit: func(record *Record) { record.PluginDisposition = "digital-service-checkout" }, want: "plugin disposition must respect"},
	}
	for _, testCase := range caseList {
		t.Run(testCase.name, func(t *testing.T) {
			record, _, err := Prototype()
			require.NoError(t, err)
			testCase.edit(&record)
			err = Validate(record)
			require.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestProjectRefusesDeclinedOffersAndSampleTraversal(t *testing.T) {
	record, samples, err := Prototype()
	require.NoError(t, err)
	record.Legal.Status = "pass"
	record.OperatorApproval = ApprovalDeclined
	_, err = Project(record, samples)
	require.ErrorContains(t, err, "declined offer cannot be projected")

	record, _, err = Prototype()
	require.NoError(t, err)
	record.Sample.Path = "../company-market-scan.md"
	require.ErrorContains(t, Validate(record), "hashed, no-signup sample")
}

func TestCheckoutURLRejectsDirectExternalTransactionLinks(t *testing.T) {
	require.NoError(t, CheckoutURL("/checkout/company-market-scan"))
	require.Error(t, CheckoutURL("https://pay.example/checkout"))
	require.Error(t, CheckoutURL("//pay.example/checkout"))
	require.Error(t, CheckoutURL("/services/company-market-scan"))
}
