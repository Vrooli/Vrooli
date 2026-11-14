package landing

import (
	"encoding/json"
	"fmt"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"landing-page-business-suite-api/internal/presentation"
)

// The domain has an ergonomic kind/content tagged union. Protobuf owns an
// explicit oneof instead. These adapters translate only that shape and the two
// typed map containers; generated descriptors enforce every other field.
// No google.protobuf.Struct or executable extension escapes this boundary.
func PresentationDocumentProto(document presentation.Document) (*shared.ProductPresentationDocument, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if err := transformPresentationPages(envelope, true); err != nil {
		return nil, err
	}
	if err := transformPresentationMaps(envelope, true); err != nil {
		return nil, err
	}
	data, err = json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	result := &shared.ProductPresentationDocument{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("encode typed presentation document: %w", err)
	}
	return result, nil
}

func PresentationDocumentFromProto(message *shared.ProductPresentationDocument) (presentation.Document, error) {
	if message == nil {
		return presentation.Document{}, fmt.Errorf("presentation document is required")
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(message)
	if err != nil {
		return presentation.Document{}, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return presentation.Document{}, err
	}
	if err := transformPresentationPages(envelope, false); err != nil {
		return presentation.Document{}, err
	}
	if err := transformPresentationMaps(envelope, false); err != nil {
		return presentation.Document{}, err
	}
	data, err = json.Marshal(envelope)
	if err != nil {
		return presentation.Document{}, err
	}
	return presentation.DecodeDocument(data)
}

func ResolvedPresentationProto(value presentation.ResolveResult) (*shared.ResolvedProductPresentation, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	page, err := transformPresentationPage(envelope["page"], true)
	if err != nil {
		return nil, err
	}
	envelope["page"] = page
	data, err = json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	result := &shared.ResolvedProductPresentation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("encode resolved presentation: %w", err)
	}
	return result, nil
}

func transformPresentationPages(envelope map[string]json.RawMessage, toWire bool) error {
	var pages []json.RawMessage
	if err := json.Unmarshal(envelope["pages"], &pages); err != nil {
		return err
	}
	for i, page := range pages {
		updated, err := transformPresentationPage(page, toWire)
		if err != nil {
			return fmt.Errorf("pages[%d]: %w", i, err)
		}
		pages[i] = updated
	}
	data, err := json.Marshal(pages)
	envelope["pages"] = data
	return err
}

func transformPresentationPage(data json.RawMessage, toWire bool) (json.RawMessage, error) {
	var page map[string]json.RawMessage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, err
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(page["blocks"], &blocks); err != nil {
		return nil, err
	}
	for i, block := range blocks {
		var kind string
		if err := json.Unmarshal(block["kind"], &kind); err != nil {
			return nil, fmt.Errorf("blocks[%d].kind: %w", i, err)
		}
		field := strings.ReplaceAll(kind, "-", "_")
		if toWire {
			wrapped, err := json.Marshal(map[string]json.RawMessage{field: block["content"]})
			if err != nil {
				return nil, err
			}
			block["content"] = wrapped
		} else {
			var content map[string]json.RawMessage
			if err := json.Unmarshal(block["content"], &content); err != nil {
				return nil, err
			}
			selected, found := content[field]
			if len(content) != 1 || !found || string(selected) == "null" {
				return nil, fmt.Errorf("blocks[%d]: kind %q does not match its typed content", i, kind)
			}
			block["content"] = selected
		}
	}
	updated, err := json.Marshal(blocks)
	if err != nil {
		return nil, err
	}
	page["blocks"] = updated
	return json.Marshal(page)
}

func transformPresentationMaps(envelope map[string]json.RawMessage, toWire bool) error {
	if raw, exists := envelope["strings"]; exists && string(raw) != "null" {
		updated, err := transformMapValues(raw, toWire)
		if err != nil {
			return fmt.Errorf("strings: %w", err)
		}
		envelope["strings"] = updated
	}
	var apps []map[string]json.RawMessage
	if err := json.Unmarshal(envelope["apps"], &apps); err != nil {
		return err
	}
	for i, app := range apps {
		var capabilities []map[string]json.RawMessage
		if raw, exists := app["capabilities"]; exists {
			if err := json.Unmarshal(raw, &capabilities); err != nil {
				return err
			}
		}
		for j, capability := range capabilities {
			if raw, exists := capability["localized_benefits"]; exists && string(raw) != "null" {
				updated, err := transformMapValues(raw, toWire)
				if err != nil {
					return fmt.Errorf("apps[%d].capabilities[%d].localized_benefits: %w", i, j, err)
				}
				capability["localized_benefits"] = updated
			}
		}
		if _, exists := app["capabilities"]; exists {
			updated, err := json.Marshal(capabilities)
			if err != nil {
				return err
			}
			app["capabilities"] = updated
		}
	}
	data, err := json.Marshal(apps)
	envelope["apps"] = data
	return err
}

func transformMapValues(data json.RawMessage, toWire bool) (json.RawMessage, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	for key, value := range values {
		if toWire {
			wrapped, err := json.Marshal(map[string]json.RawMessage{"values": value})
			if err != nil {
				return nil, err
			}
			values[key] = wrapped
		} else {
			var wrapped map[string]json.RawMessage
			if err := json.Unmarshal(value, &wrapped); err != nil {
				return nil, err
			}
			if len(wrapped) != 1 || wrapped["values"] == nil {
				return nil, fmt.Errorf("invalid typed map value for %q", key)
			}
			values[key] = wrapped["values"]
		}
	}
	return json.Marshal(values)
}
