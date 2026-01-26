package ingest

import (
	"context"
	"testing"
)

func TestUpsertRecordValidate(t *testing.T) {
	req := UpsertRecordRequest{}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}

	req = UpsertRecordRequest{
		Namespace:  "ns",
		Collection: "coll",
		RecordID:   "rec",
		Content:    "hello",
		Visibility: "public",
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteRecordGuards(t *testing.T) {
	service := &Service{}
	if _, err := service.DeleteRecord(context.Background(), ""); err == nil {
		t.Fatalf("expected record id error")
	}
	if _, err := service.DeleteRecord(context.Background(), "rec"); err == nil {
		t.Fatalf("expected dependency error")
	}
}

func TestContentHashStable(t *testing.T) {
	one := contentHash("ns", "hello\r\nworld")
	two := contentHash("ns", "hello\nworld")
	if one != two {
		t.Fatalf("expected normalized hash to match")
	}
}
