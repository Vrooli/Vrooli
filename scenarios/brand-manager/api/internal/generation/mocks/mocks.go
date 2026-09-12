// Package mocks provides in-memory fakes for the generation domain seams so
// service and handler unit tests run without reaching out to a real AI provider,
// image-tools, or the brands/assets domains. Mirrors internal/assets/mocks.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"brand-manager/internal/generation"
)

// FakeProviders satisfies generation.Providers. It records the text requests it
// receives and returns scripted responses, so a test can assert the prompts the
// service built and drive the success/failure/unavailable paths.
type FakeProviders struct {
	AvailableValue bool
	StatusValues   []generation.ProviderStatus

	// TextResponder scripts the provider response. When nil, the fake returns a
	// benign empty-JSON text response.
	TextResponder func(req generation.TextRequest) (generation.TextResponse, error)

	TextCalls atomic.Int64

	mu       sync.Mutex
	textReqs []generation.TextRequest
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

// TextRequests returns a copy of the text requests the fake received.
func (f *FakeProviders) TextRequests() []generation.TextRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.TextRequest(nil), f.textReqs...)
}

var _ generation.Providers = (*FakeProviders)(nil)

// FakeImageBackend satisfies generation.ImageBackend. Each operation records its
// request and returns a scripted ImageOutput (or a benign tiny PNG by default),
// so service tests drive the image path without a real image-tools.
type FakeImageBackend struct {
	StatusValue generation.ImageBackendStatus

	GenerateResponder func(req generation.ImageGenerateRequest) (generation.ImageOutput, error)
	EditResponder     func(req generation.ImageEditRequest) (generation.ImageOutput, error)
	RemoveResponder   func(req generation.ImageRemoveBackgroundRequest) (generation.ImageOutput, error)
	ResizeResponder   func(src []byte, w, h int) (generation.ImageOutput, error)
	FlattenResponder  func(src []byte, w, h int, bg string) (generation.ImageOutput, error)

	mu          sync.Mutex
	generateIn  []generation.ImageGenerateRequest
	editIn      []generation.ImageEditRequest
	removeIn    []generation.ImageRemoveBackgroundRequest
	resizeCalls int
	flattenBGs  []string
}

func defaultPNG() generation.ImageOutput {
	return generation.ImageOutput{Data: []byte("\x89PNG\r\n\x1a\n"), MimeType: "image/png", ModelID: "fake-image", Tier: "local-gpu"}
}

func (f *FakeImageBackend) Status(context.Context) generation.ImageBackendStatus {
	return f.StatusValue
}

func (f *FakeImageBackend) Generate(_ context.Context, req generation.ImageGenerateRequest) (generation.ImageOutput, error) {
	f.mu.Lock()
	f.generateIn = append(f.generateIn, req)
	f.mu.Unlock()
	if f.GenerateResponder != nil {
		return f.GenerateResponder(req)
	}
	return defaultPNG(), nil
}

func (f *FakeImageBackend) Edit(_ context.Context, req generation.ImageEditRequest) (generation.ImageOutput, error) {
	f.mu.Lock()
	f.editIn = append(f.editIn, req)
	f.mu.Unlock()
	if f.EditResponder != nil {
		return f.EditResponder(req)
	}
	return defaultPNG(), nil
}

func (f *FakeImageBackend) RemoveBackground(_ context.Context, req generation.ImageRemoveBackgroundRequest) (generation.ImageOutput, error) {
	f.mu.Lock()
	f.removeIn = append(f.removeIn, req)
	f.mu.Unlock()
	if f.RemoveResponder != nil {
		return f.RemoveResponder(req)
	}
	return defaultPNG(), nil
}

func (f *FakeImageBackend) Resize(_ context.Context, src []byte, w, h int) (generation.ImageOutput, error) {
	f.mu.Lock()
	f.resizeCalls++
	f.mu.Unlock()
	if f.ResizeResponder != nil {
		return f.ResizeResponder(src, w, h)
	}
	out := defaultPNG()
	out.Tier = "deterministic"
	out.ModelID = ""
	return out, nil
}

func (f *FakeImageBackend) Flatten(_ context.Context, src []byte, w, h int, bg string) (generation.ImageOutput, error) {
	f.mu.Lock()
	f.flattenBGs = append(f.flattenBGs, bg)
	f.mu.Unlock()
	if f.FlattenResponder != nil {
		return f.FlattenResponder(src, w, h, bg)
	}
	out := defaultPNG()
	out.Tier = "deterministic"
	out.ModelID = ""
	return out, nil
}

// GenerateRequests / EditRequests / RemoveRequests / FlattenBackgrounds return
// copies of the recorded calls for assertions.
func (f *FakeImageBackend) GenerateRequests() []generation.ImageGenerateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.ImageGenerateRequest(nil), f.generateIn...)
}

func (f *FakeImageBackend) EditRequests() []generation.ImageEditRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.ImageEditRequest(nil), f.editIn...)
}

func (f *FakeImageBackend) RemoveRequests() []generation.ImageRemoveBackgroundRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]generation.ImageRemoveBackgroundRequest(nil), f.removeIn...)
}

func (f *FakeImageBackend) FlattenBackgrounds() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.flattenBGs...)
}

var _ generation.ImageBackend = (*FakeImageBackend)(nil)

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
// keyed by (brand_id, filename) so Exists/Read work like the real upsert, and
// returns a deterministic asset id per filename.
type FakeAssetStore struct {
	mu       sync.Mutex
	Stored   []generation.AssetUpload
	byKey    map[string]generation.AssetUpload // brandID|filename -> upload
	byID     map[string]generation.AssetUpload // assetID -> upload
	StoreErr error
	ReadErr  error
}

func assetKey(brandID, filename string) string { return brandID + "|" + filename }

func (f *FakeAssetStore) Store(_ context.Context, in generation.AssetUpload) (generation.StoredAsset, error) {
	if f.StoreErr != nil {
		return generation.StoredAsset{}, f.StoreErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byKey == nil {
		f.byKey = map[string]generation.AssetUpload{}
		f.byID = map[string]generation.AssetUpload{}
	}
	f.Stored = append(f.Stored, in)
	id := "asset-" + in.Filename
	f.byKey[assetKey(in.BrandID, in.Filename)] = in
	f.byID[id] = in
	return generation.StoredAsset{
		ID:       id,
		Filename: in.Filename,
		MimeType: in.MimeType,
		Size:     int64(len(in.Content)),
	}, nil
}

func (f *FakeAssetStore) Read(_ context.Context, assetID string) (generation.AssetBytes, error) {
	if f.ReadErr != nil {
		return generation.AssetBytes{}, f.ReadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.byID[assetID]
	if !ok {
		return generation.AssetBytes{}, generation.ErrSourceAssetNotFound{ID: assetID}
	}
	return generation.AssetBytes{ID: assetID, Filename: in.Filename, MimeType: in.MimeType, Content: in.Content}, nil
}

func (f *FakeAssetStore) Exists(_ context.Context, brandID, filename string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.byKey[assetKey(brandID, filename)]
	return ok, nil
}

// SeedAsset registers a source asset so Read returns it.
func (f *FakeAssetStore) SeedAsset(assetID, brandID, filename, mime string, content []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byKey == nil {
		f.byKey = map[string]generation.AssetUpload{}
		f.byID = map[string]generation.AssetUpload{}
	}
	up := generation.AssetUpload{BrandID: brandID, Filename: filename, MimeType: mime, Content: content}
	f.byKey[assetKey(brandID, filename)] = up
	f.byID[assetID] = up
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
	ProviderStatusFunc     func(ctx context.Context) (bool, []generation.ProviderStatus)
	ImageBackendStatusFunc func(ctx context.Context) generation.ImageBackendStatus
	GenerateElementsFunc   func(ctx context.Context, brandID string, elements []string, model string) (generation.ElementsResult, error)
	GenerateImageFunc      func(ctx context.Context, in generation.GenerateImageInput) (generation.ImageResult, error)
	EditImageFunc          func(ctx context.Context, in generation.EditImageInput) (generation.ImageResult, error)
	RemoveBackgroundFunc   func(ctx context.Context, in generation.RemoveBackgroundInput) (generation.ImageResult, error)
	DeriveIconsFunc        func(ctx context.Context, in generation.DeriveIconsInput) ([]generation.ImageResult, []string, error)
}

func (f FakeService) ProviderStatus(ctx context.Context) (bool, []generation.ProviderStatus) {
	if f.ProviderStatusFunc != nil {
		return f.ProviderStatusFunc(ctx)
	}
	return false, nil
}

func (f FakeService) ImageBackendStatus(ctx context.Context) generation.ImageBackendStatus {
	if f.ImageBackendStatusFunc != nil {
		return f.ImageBackendStatusFunc(ctx)
	}
	return generation.ImageBackendStatus{}
}

func (f FakeService) GenerateElements(ctx context.Context, brandID string, elements []string, model string) (generation.ElementsResult, error) {
	if f.GenerateElementsFunc != nil {
		return f.GenerateElementsFunc(ctx, brandID, elements, model)
	}
	return generation.ElementsResult{}, nil
}

func (f FakeService) GenerateImage(ctx context.Context, in generation.GenerateImageInput) (generation.ImageResult, error) {
	if f.GenerateImageFunc != nil {
		return f.GenerateImageFunc(ctx, in)
	}
	return generation.ImageResult{}, nil
}

func (f FakeService) EditImage(ctx context.Context, in generation.EditImageInput) (generation.ImageResult, error) {
	if f.EditImageFunc != nil {
		return f.EditImageFunc(ctx, in)
	}
	return generation.ImageResult{}, nil
}

func (f FakeService) RemoveBackground(ctx context.Context, in generation.RemoveBackgroundInput) (generation.ImageResult, error) {
	if f.RemoveBackgroundFunc != nil {
		return f.RemoveBackgroundFunc(ctx, in)
	}
	return generation.ImageResult{}, nil
}

func (f FakeService) DeriveIcons(ctx context.Context, in generation.DeriveIconsInput) ([]generation.ImageResult, []string, error) {
	if f.DeriveIconsFunc != nil {
		return f.DeriveIconsFunc(ctx, in)
	}
	return nil, nil, nil
}

var _ generation.Service = (*FakeService)(nil)
