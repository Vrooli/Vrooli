package designs

import (
	"fmt"
	"strings"

	"react-component-library/internal/imageheader"
)

// MaxAssetBytes bounds one decoded upload. Reference images are screenshots; a
// full-resolution capture fits comfortably below 8 MiB.
const MaxAssetBytes = imageheader.MaxBytes

// Asset fidelities describe how faithful an upload is to the rendered page.
const (
	FidelityReference = "reference"
	FidelityCaptured  = "captured"
)

func normalizeAssetInput(input AssetInput) (AssetInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return AssetInput{}, fmt.Errorf("%w: asset name is required", ErrInvalid)
	}
	input.Mime = imageheader.NormalizeMime(input.Mime)
	if _, ok := imageheader.AllowedMimes[input.Mime]; !ok {
		return AssetInput{}, fmt.Errorf("%w: asset mime must be image/png, image/jpeg, image/webp, image/gif or image/svg+xml", ErrInvalid)
	}
	if len(input.Data) == 0 {
		return AssetInput{}, fmt.Errorf("%w: asset data is required", ErrInvalid)
	}
	if len(input.Data) > MaxAssetBytes {
		return AssetInput{}, fmt.Errorf("%w: asset exceeds the 8 MiB limit", ErrInvalid)
	}
	input.Fidelity = strings.ToLower(strings.TrimSpace(input.Fidelity))
	if input.Fidelity == "" {
		input.Fidelity = FidelityReference
	}
	if input.Fidelity != FidelityReference && input.Fidelity != FidelityCaptured {
		return AssetInput{}, fmt.Errorf("%w: asset fidelity must be reference or captured", ErrInvalid)
	}
	return input, nil
}

// imageDimensions is kept as the package-local name the asset reader already
// uses; the parser lives in internal/imageheader so the workspace store shares
// exactly the same one.
func imageDimensions(mime string, data []byte) (int, int) {
	return imageheader.Dimensions(mime, data)
}
