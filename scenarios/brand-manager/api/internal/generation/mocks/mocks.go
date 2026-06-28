// Package mocks provides in-memory fakes for the generation domain seams so
// service and handler unit tests run without reaching out to a real AI provider
// or the brands/assets domains. Mirrors internal/assets/mocks.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"brand-manager/internal/generation"
)

// FakeProviders satisfies generation.Providers. It records the requests it
// receives and returns scripted responses, so a test can assert the prompts the
// service built and drive the success/failure/unavailable paths.
type FakeProviders struct {
	AvailableValue bool
	StatusValues   []generation.ProviderStatus

	// TextResponder / ImageResponder script the provider responses. When nil, the
	// fake returns a benign empty-JSON text response / a tiny PNG image response.
	TextResponder  func(req generation.TextRequest) (generation.TextResponse, error)
	ImageResponder func(req generation.ImageRequest) (generation.ImageResponse, error)

	TextCalls  atomic.Int64
	ImageCalls atomic.Int64

	mu       sync.Mutex
	textReqs []generation.TextRequest
	imgReqs  []generation.ImageRequest
}

func (f *FakeProviders) Available(context.Context) bool { return f.AvailableValue }

func (f *FakeProviders) Statuses(context.Context) []generation.ProviderStatus {
	return f.StatusValues
}

func (f *FakeProviders) GenerateText(_ context.Context, req generation.TextRequest) (generation.TextResponse, error) {
	f.TextCalls.Add(1)
	f.mu.Lock()
	f.textReqs = append(f.textReqs, req)
	f.mu.Unlock()
	if f.TextResponder != nil {
		return f.TextResponder(req)
	}
	return generation.TextResponse{Text: "{}", Provider: "fake", Model: "fake-model"}, nil
}

func (f *FakeProviders) GenerateImage(_ context.Context, req generation.ImageRequest) (generation.ImageResponse, error) {
	f.ImageCalls.Add(1)
	f.mu.Lock()
	f.imgReqs = append(f.imgReqs, req)
	f.mu.Unlock()
	if f.ImageResponder != nil {
		return f.ImageResponder(req)
	}
	return generation.ImageResponse{Data: []byte("\x89PNG"), MimeType: "image/png", Provider: "fake", Model: "fake-image"}, nil
}

// TextRequests returns a copy of the text requests the fake received.
func (f *FakeProviders) TextRequests() []generation.TextRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.TextRequest(nil), f.textReqs...)
}

// ImageRequests returns a copy of the image requests the fake received.
func (f *FakeProviders) ImageRequests() []generation.ImageRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.ImageRequest(nil), f.imgReqs...)
}

var _ generation.Providers = (*FakeProviders)(nil)

// FakeBrandStore satisfies generation.BrandStore. Known holds the brands that
// exist (keyed by id); ApplyElements records the merged facets and bumps a
// per-brand version counter.
type FakeBrandStore struct {
	mu       sync.Mutex
	Known    map[string]generation.BrandView
	Applied  []generation.ApplyElementsInput
	GetErr   error
	ApplyErr error
}

func (f *FakeBrandStore) Get(_ context.Context, brandID string) (generation.BrandView, error) {
	if f.GetErr != nil {
		return generation.BrandView{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Known[brandID]
	if !ok {
		return generation.BrandView{}, generation.ErrBrandNotFound{ID: brandID}
	}
	return b, nil
}

func (f *FakeBrandStore) ApplyElements(_ context.Context, in generation.ApplyElementsInput) (int, error) {
	if f.ApplyErr != nil {
		return 0, f.ApplyErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Applied = append(f.Applied, in)
	b := f.Known[in.BrandID]
	b.Version++
	f.Known[in.BrandID] = b
	return b.Version, nil
}

// Seed registers a brand so Get returns it.
func (f *FakeBrandStore) Seed(b generation.BrandView) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Known == nil {
		f.Known = map[string]generation.BrandView{}
	}
	f.Known[b.ID] = b
}

// AppliedInputs returns a copy of the ApplyElements calls the fake recorded.
func (f *FakeBrandStore) AppliedInputs() []generation.ApplyElementsInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.ApplyElementsInput(nil), f.Applied...)
}

var _ generation.BrandStore = (*FakeBrandStore)(nil)

// FakeAssetStore satisfies generation.AssetStore. It records each stored upload
// and returns a deterministic asset id derived from the upload count.
type FakeAssetStore struct {
	mu       sync.Mutex
	Stored   []generation.AssetUpload
	StoreErr error
}

func (f *FakeAssetStore) Store(_ context.Context, in generation.AssetUpload) (generation.StoredAsset, error) {
	if f.StoreErr != nil {
		return generation.StoredAsset{}, f.StoreErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Stored = append(f.Stored, in)
	return generation.StoredAsset{
		ID:       "asset-stored",
		Filename: in.Filename,
		MimeType: in.MimeType,
		Size:     int64(len(in.Content)),
	}, nil
}

// StoredUploads returns a copy of the uploads the fake recorded.
func (f *FakeAssetStore) StoredUploads() []generation.AssetUpload {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.AssetUpload(nil), f.Stored...)
}

var _ generation.AssetStore = (*FakeAssetStore)(nil)

// FakeService satisfies generation.Service for handler tests that want to drive
// the transport edge without the orchestration logic. The Func fields override
// per-method behaviour; nil fields return zero values.
type FakeService struct {
	ProviderStatusFunc   func(ctx context.Context) (bool, []generation.ProviderStatus)
	GenerateElementsFunc func(ctx context.Context, brandID string, elements []string, model string) (generation.ElementsResult, error)
	GenerateImageFunc    func(ctx context.Context, brandID, imageType, model string) (generation.ImageResult, error)
}

func (f FakeService) ProviderStatus(ctx context.Context) (bool, []generation.ProviderStatus) {
	if f.ProviderStatusFunc != nil {
		return f.ProviderStatusFunc(ctx)
	}
	return false, nil
}

func (f FakeService) GenerateElements(ctx context.Context, brandID string, elements []string, model string) (generation.ElementsResult, error) {
	if f.GenerateElementsFunc != nil {
		return f.GenerateElementsFunc(ctx, brandID, elements, model)
	}
	return generation.ElementsResult{}, nil
}

func (f FakeService) GenerateImage(ctx context.Context, brandID, imageType, model string) (generation.ImageResult, error) {
	if f.GenerateImageFunc != nil {
		return f.GenerateImageFunc(ctx, brandID, imageType, model)
	}
	return generation.ImageResult{}, nil
}

var _ generation.Service = (*FakeService)(nil)
