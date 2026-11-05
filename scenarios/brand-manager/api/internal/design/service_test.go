package design_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"brand-manager/internal/design"
	mocks "brand-manager/internal/design/mocks"
)

func TestService_GenerateRendersBrand(t *testing.T) {
	store := &mocks.FakeBrandStore{}
	store.Seed(design.Brand{
		ID:      "b1",
		Name:    "Acme",
		Version: 2,
		Colors:  design.Colors{Primary: "#112233"},
	})
	svc := design.NewService(store, nil)

	got, err := svc.GenerateDesignLanguage(context.Background(), "b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BrandID != "b1" {
		t.Errorf("BrandID = %q, want b1", got.BrandID)
	}
	if !strings.Contains(got.Markdown, "# Acme DESIGN.md") {
		t.Error("markdown missing brand title")
	}
	if !strings.Contains(got.Markdown, "#112233") {
		t.Error("markdown missing primary color")
	}
}

func TestService_GenerateTrimsAndRejectsBlankID(t *testing.T) {
	svc := design.NewService(&mocks.FakeBrandStore{}, nil)

	_, err := svc.GenerateDesignLanguage(context.Background(), "   ")
	var invalid design.ErrInvalidDesign
	if !errors.As(err, &invalid) {
		t.Fatalf("expected ErrInvalidDesign, got %v", err)
	}
}

func TestService_GenerateUnknownBrandIsNotFound(t *testing.T) {
	svc := design.NewService(&mocks.FakeBrandStore{}, nil)

	_, err := svc.GenerateDesignLanguage(context.Background(), "ghost")
	var notFound design.ErrBrandNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ErrBrandNotFound, got %v", err)
	}
}

func TestService_GeneratePropagatesLookupError(t *testing.T) {
	store := &mocks.FakeBrandStore{GetErr: errors.New("db down")}
	svc := design.NewService(store, nil)

	_, err := svc.GenerateDesignLanguage(context.Background(), "b1")
	if err == nil {
		t.Fatal("expected the lookup error to propagate")
	}
	var notFound design.ErrBrandNotFound
	if errors.As(err, &notFound) {
		t.Error("a genuine lookup failure must not masquerade as not-found")
	}
}
