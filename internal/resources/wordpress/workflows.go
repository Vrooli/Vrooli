package wordpress

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

const MaxWorkflowPayloadBytes = 64 << 10

type CanonicalFormEvent struct {
	Provider string `json:"provider"`
	FormID   string `json:"formId"`
	EventID  string `json:"eventId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

type SubmissionDisposition string

const (
	SubmissionCreated   SubmissionDisposition = "created"
	SubmissionDuplicate SubmissionDisposition = "duplicate"
	SubmissionConflict  SubmissionDisposition = "identity_conflict"
)

type formWire struct {
	FormID     string         `json:"form_id"`
	EventID    string         `json:"event_id"`
	PostedData map[string]any `json:"posted_data"`
	FormData   struct {
		ID any `json:"id"`
	} `json:"form_data"`
	EntryID int             `json:"entry_id"`
	Fields  json.RawMessage `json:"fields"`
	Data    map[string]any  `json:"data"`
}

type wpFormsField struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// NormalizeFormSubmission maps the three selected free plugins and the
// explicitly synthetic Elementor-shaped contract into one validated event.
// Every provider must carry a hidden submission event ID; WPForms Lite's
// entry_id is zero and is not used as a retry identity.
func NormalizeFormSubmission(provider string, data []byte) (CanonicalFormEvent, error) {
	if len(data) == 0 || len(data) > MaxWorkflowPayloadBytes {
		return CanonicalFormEvent{}, fmt.Errorf("form payload must be 1..%d bytes", MaxWorkflowPayloadBytes)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var wire formWire
	if err := decoder.Decode(&wire); err != nil {
		return CanonicalFormEvent{}, fmt.Errorf("decode %s form payload: %w", provider, err)
	}
	values := map[string]string{}
	formID := strings.TrimSpace(wire.FormID)
	formValue := func(value any) string {
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	switch provider {
	case "contact-form-7":
		for key, value := range wire.PostedData {
			values[key] = formValue(value)
		}
	case "wpforms-lite":
		if formID == "" {
			formID = formValue(wire.FormData.ID)
		}
		var fields []wpFormsField
		if err := json.Unmarshal(wire.Fields, &fields); err != nil {
			return CanonicalFormEvent{}, fmt.Errorf("decode WPForms Lite fields: %w", err)
		}
		for _, field := range fields {
			if field.Name == "" || values[field.Name] != "" {
				return CanonicalFormEvent{}, fmt.Errorf("WPForms Lite payload has a missing or repeated field name")
			}
			values[field.Name] = formValue(field.Value)
		}
	case "forminator":
		for key, value := range wire.Data {
			values[key] = formValue(value)
		}
		if values["event_id"] == "" {
			values["event_id"] = values["hidden-1"]
		}
		if values["full-name"] == "" {
			values["full-name"] = values["name-1"]
		}
		if values["email"] == "" {
			values["email"] = values["email-1"]
		}
		if values["message"] == "" {
			values["message"] = values["textarea-1"]
		}
	case "elementor-shaped-synthetic":
		var shaped struct {
			FormID string `json:"form_id"`
			Fields map[string]struct {
				Value any `json:"value"`
			} `json:"fields"`
		}
		if err := json.Unmarshal(data, &shaped); err != nil {
			return CanonicalFormEvent{}, err
		}
		if formID == "" {
			formID = shaped.FormID
		}
		for key, field := range shaped.Fields {
			values[key] = formValue(field.Value)
		}
	default:
		return CanonicalFormEvent{}, fmt.Errorf("unsupported form provider %q", provider)
	}
	if formID == "" {
		return CanonicalFormEvent{}, fmt.Errorf("form submission is missing form_id")
	}
	if values["event_id"] == "" {
		values["event_id"] = wire.EventID
	}
	name, email, message := values["full-name"], values["email"], values["message"]
	if name == "" {
		name = values["name"]
	}
	if strings.TrimSpace(values["event_id"]) == "" {
		return CanonicalFormEvent{}, fmt.Errorf("form submission is missing its hidden event_id retry token")
	}
	if name == "" {
		return CanonicalFormEvent{}, fmt.Errorf("missing_name")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return CanonicalFormEvent{}, fmt.Errorf("invalid_email")
	}
	if message == "" {
		return CanonicalFormEvent{}, fmt.Errorf("missing_message")
	}
	if utf8.RuneCountInString(name) > 200 || utf8.RuneCountInString(email) > 320 || utf8.RuneCountInString(message) > 4000 {
		return CanonicalFormEvent{}, fmt.Errorf("form field exceeds its accepted size")
	}
	return CanonicalFormEvent{Provider: provider, FormID: formID, EventID: values["event_id"], Name: name, Email: email, Message: message}, nil
}

type SubmissionLedger struct {
	mu            sync.Mutex
	payloadHashes map[string]string
}

func NewSubmissionLedger() *SubmissionLedger {
	return &SubmissionLedger{payloadHashes: map[string]string{}}
}

func (l *SubmissionLedger) Accept(event CanonicalFormEvent) (SubmissionDisposition, error) {
	if event.Provider == "" || event.FormID == "" || event.EventID == "" {
		return "", fmt.Errorf("form idempotency identity is incomplete")
	}
	key := event.Provider + ":" + event.FormID + ":" + event.EventID
	encoded, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.payloadHashes[key]; ok {
		if existing == hash {
			return SubmissionDuplicate, nil
		}
		return SubmissionConflict, nil
	}
	l.payloadHashes[key] = hash
	return SubmissionCreated, nil
}

type CanonicalOrder struct {
	SiteID        string      `json:"siteId"`
	OrderID       string      `json:"orderId"`
	Revision      string      `json:"revision"`
	Status        string      `json:"status"`
	CustomerEmail string      `json:"customerEmail"`
	Items         []OrderItem `json:"items"`
}

type OrderItem struct {
	ProductID int64  `json:"productId"`
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	Total     string `json:"total"`
}

type wooOrderWire struct {
	ID          any    `json:"id"`
	Status      string `json:"status"`
	ModifiedGMT string `json:"date_modified_gmt"`
	Billing     struct {
		Email string `json:"email"`
	} `json:"billing"`
	Items []struct {
		ProductID int64  `json:"product_id"`
		SKU       string `json:"sku"`
		Quantity  int    `json:"quantity"`
		Total     string `json:"total"`
	} `json:"line_items"`
}

func NormalizeWooOrder(siteID string, payload []byte, allowedSKUs map[string]bool) (CanonicalOrder, error) {
	if err := validateSiteID(siteID); err != nil {
		return CanonicalOrder{}, err
	}
	if len(payload) == 0 || len(payload) > MaxWorkflowPayloadBytes {
		return CanonicalOrder{}, fmt.Errorf("WooCommerce order payload must be 1..%d bytes", MaxWorkflowPayloadBytes)
	}
	var wire wooOrderWire
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	if err := decoder.Decode(&wire); err != nil {
		return CanonicalOrder{}, fmt.Errorf("decode WooCommerce order: %w", err)
	}
	orderID := strings.TrimSpace(fmt.Sprint(wire.ID))
	if orderID == "" || orderID == "<nil>" {
		return CanonicalOrder{}, fmt.Errorf("missing_order_id")
	}
	if wire.ModifiedGMT == "" {
		return CanonicalOrder{}, fmt.Errorf("missing_revision")
	}
	switch wire.Status {
	case "pending", "processing", "on-hold", "completed", "cancelled", "refunded", "failed":
	default:
		return CanonicalOrder{}, fmt.Errorf("unsupported_order_status")
	}
	if _, err := mail.ParseAddress(wire.Billing.Email); err != nil {
		return CanonicalOrder{}, fmt.Errorf("invalid_customer_email")
	}
	if len(wire.Items) == 0 {
		return CanonicalOrder{}, fmt.Errorf("missing_line_items")
	}
	items := make([]OrderItem, 0, len(wire.Items))
	for _, item := range wire.Items {
		if !allowedSKUs[item.SKU] {
			return CanonicalOrder{}, fmt.Errorf("unknown_sku: %s", item.SKU)
		}
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return CanonicalOrder{}, fmt.Errorf("invalid_line_item")
		}
		items = append(items, OrderItem{ProductID: item.ProductID, SKU: item.SKU, Quantity: item.Quantity, Total: item.Total})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SKU == items[j].SKU {
			return items[i].ProductID < items[j].ProductID
		}
		return items[i].SKU < items[j].SKU
	})
	return CanonicalOrder{SiteID: siteID, OrderID: orderID, Revision: wire.ModifiedGMT, Status: wire.Status, CustomerEmail: wire.Billing.Email, Items: items}, nil
}

type OrderLedger struct {
	mu            sync.Mutex
	payloadHashes map[string]string
}

func NewOrderLedger() *OrderLedger { return &OrderLedger{payloadHashes: map[string]string{}} }

func (l *OrderLedger) Accept(order CanonicalOrder) (SubmissionDisposition, error) {
	if order.SiteID == "" || order.OrderID == "" || order.Revision == "" {
		return "", fmt.Errorf("order idempotency identity is incomplete")
	}
	key := order.SiteID + ":" + order.OrderID + ":" + order.Revision
	encoded, err := json.Marshal(order)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.payloadHashes[key]; ok {
		if existing == hash {
			return SubmissionDuplicate, nil
		}
		return SubmissionConflict, nil
	}
	l.payloadHashes[key] = hash
	return SubmissionCreated, nil
}

// ContentUpdateLedger models safe retry handling for a custom-field content
// update job. It tracks a post's current synthetic fields and the exact patch
// identity, returning duplicate only when an identical update is replayed.
type ContentUpdateLedger struct {
	mu           sync.Mutex
	posts        map[string]map[string]any
	updateHashes map[string]string
}

// ContentMigrationLedger models destination-side source identity and content
// hash handling for the synthetic-only cross-site migration ground.
type ContentMigrationLedger struct {
	mu      sync.Mutex
	entries map[string]ContentMigrationRecord
}

type ContentMigrationDisposition string

const (
	ContentMigrationCreated   ContentMigrationDisposition = "created"
	ContentMigrationUpdated   ContentMigrationDisposition = "updated"
	ContentMigrationDuplicate ContentMigrationDisposition = "duplicate"
	ContentMigrationConflict  ContentMigrationDisposition = "source_conflict"
)

func NewContentMigrationLedger() *ContentMigrationLedger {
	return &ContentMigrationLedger{entries: make(map[string]ContentMigrationRecord)}
}

func (l *ContentMigrationLedger) Plan(targetSiteID string, record ContentMigrationRecord) (ContentMigrationDisposition, error) {
	if err := validateContentMigrationRecord(targetSiteID, record); err != nil {
		return "", err
	}
	key := targetSiteID + "\x00" + record.Slug
	l.mu.Lock()
	defer l.mu.Unlock()
	existing, ok := l.entries[key]
	if !ok {
		return ContentMigrationCreated, nil
	}
	if existing.SourceKey != record.SourceKey {
		return ContentMigrationConflict, nil
	}
	if existing.ContentHash == record.ContentHash {
		return ContentMigrationDuplicate, nil
	}
	return ContentMigrationUpdated, nil
}

func (l *ContentMigrationLedger) Apply(targetSiteID string, record ContentMigrationRecord) (ContentMigrationDisposition, error) {
	if err := validateContentMigrationRecord(targetSiteID, record); err != nil {
		return "", err
	}
	key := targetSiteID + "\x00" + record.Slug
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.entries[key]; ok {
		if existing.SourceKey != record.SourceKey {
			return ContentMigrationConflict, nil
		}
		if existing.ContentHash == record.ContentHash {
			return ContentMigrationDuplicate, nil
		}
		l.entries[key] = record
		return ContentMigrationUpdated, nil
	}
	l.entries[key] = record
	return ContentMigrationCreated, nil
}

func validateContentMigrationRecord(targetSiteID string, record ContentMigrationRecord) error {
	sourceSiteID := strings.SplitN(record.SourceKey, "/", 2)[0]
	if !contentMigrationSiteIDPattern.MatchString(targetSiteID) || !contentSlugPattern.MatchString(record.Slug) || !contentMigrationSiteIDPattern.MatchString(sourceSiteID) || record.SourceKey != sourceSiteID+"/"+record.Slug {
		return errors.New("content migration target, source key or slug is invalid")
	}
	if record.Categories == nil || record.Tags == nil {
		return errors.New("content migration categories and tags must be explicit arrays")
	}
	if err := validateCoreContentPost(map[string]any{"slug": record.Slug, "title": record.Title, "content": record.Content, "excerpt": record.Excerpt, "categories": record.Categories, "tags": record.Tags, "status": record.Status, "client_type": record.ClientType, "outcome_metric": float64(record.OutcomeMetric), "verified_on": record.VerifiedOn}); err != nil {
		return err
	}
	expectedHash, err := contentMigrationHash(record)
	if err != nil || record.ContentHash != expectedHash {
		return errors.New("content migration hash does not match canonical content")
	}
	return nil
}

func contentMigrationHash(record ContentMigrationRecord) (string, error) {
	categories := append([]string(nil), record.Categories...)
	tags := append([]string(nil), record.Tags...)
	sort.Strings(categories)
	sort.Strings(tags)
	canonical, err := json.Marshal(map[string]any{"slug": record.Slug, "title": record.Title, "content": record.Content, "excerpt": record.Excerpt, "categories": categories, "tags": tags, "status": record.Status, "client_type": record.ClientType, "outcome_metric": record.OutcomeMetric, "verified_on": record.VerifiedOn})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

type ContentUpdateDisposition string

const (
	ContentCreated   ContentUpdateDisposition = "created"
	ContentUpdated   ContentUpdateDisposition = "updated"
	ContentDuplicate ContentUpdateDisposition = "duplicate"
	ContentConflict  ContentUpdateDisposition = "identity_conflict"
)

func NewContentUpdateLedger() *ContentUpdateLedger {
	return &ContentUpdateLedger{posts: map[string]map[string]any{}, updateHashes: map[string]string{}}
}

func (l *ContentUpdateLedger) CreateIfMissing(postID string, fields map[string]any) (bool, error) {
	if strings.TrimSpace(postID) == "" || len(fields) == 0 {
		return false, fmt.Errorf("content post identity and initial fields are required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.posts[postID]; exists {
		return false, nil
	}
	l.posts[postID] = cloneContentFields(fields)
	return true, nil
}

func (l *ContentUpdateLedger) Apply(postID, updateID string, patch map[string]any) (ContentUpdateDisposition, map[string]any, error) {
	if strings.TrimSpace(postID) == "" || strings.TrimSpace(updateID) == "" || len(patch) == 0 {
		return "", nil, fmt.Errorf("content update identity and patch are required")
	}
	encoded, err := json.Marshal(patch)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(encoded)
	hash := hex.EncodeToString(digest[:])
	key := postID + "\x00" + updateID
	l.mu.Lock()
	defer l.mu.Unlock()
	post, exists := l.posts[postID]
	if !exists {
		return "", nil, fmt.Errorf("content post %q does not exist", postID)
	}
	if prior, exists := l.updateHashes[key]; exists {
		if prior == hash {
			return ContentDuplicate, cloneContentFields(post), nil
		}
		return ContentConflict, cloneContentFields(post), nil
	}
	for field, value := range patch {
		post[field] = value
	}
	l.updateHashes[key] = hash
	return ContentUpdated, cloneContentFields(post), nil
}

func cloneContentFields(fields map[string]any) map[string]any {
	clone := make(map[string]any, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}
