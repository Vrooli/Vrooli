package resources

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

// ResourceDataInspection discovers strong byte pins for bounded non-executable
// upstream metadata. Only the resource's declared credential may be selected.
type ResourceDataInspection struct {
	URL         string
	GitBlobSHA1 string
	MaxBytes    int64
	Destination string
	LogicalID   string
	Field       string
}

func (c *Controller) InspectData(ctx context.Context, name string, request ResourceDataInspection) (binaryfetch.DataInspectionReceipt, error) {
	var empty binaryfetch.DataInspectionReceipt
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return empty, errors.New("inspection requires one resource name")
	}
	manifest, err := manifestpkg.LoadForDataInspection(filepath.Join(c.Root, "resources", name, "resource.json"))
	if err != nil {
		return empty, err
	}
	if manifest.Name != name {
		return empty, errors.New("inspection resource identity does not match manifest")
	}
	var target binaryfetch.AcquisitionTarget
	matched := false
	for _, data := range manifest.ManagedService.DataArtifacts {
		if data.Acquisition == nil {
			continue
		}
		for _, candidate := range data.Acquisition.Targets {
			if data.Acquisition.EffectiveKind(candidate) == "url" && candidate.URL == request.URL && candidate.Auth != nil && candidate.Auth.Scheme == "bearer" && candidate.Auth.LogicalID == request.LogicalID && candidate.Auth.Field == request.Field {
				target = candidate
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}
	if !matched {
		return empty, errors.New("inspection URL and credential must exactly match a declared data artifact target")
	}
	auth, err := resolveAcquisitionAuthorization(ctx, manifest, target)
	if err != nil {
		return empty, err
	}
	return binaryfetch.InspectData(ctx, binaryfetch.DataInspection{URL: request.URL, GitBlobSHA1: request.GitBlobSHA1, MaxBytes: request.MaxBytes, Authorization: auth}, request.Destination)
}

func resolveAcquisitionAuthorization(ctx context.Context, manifest ResourceManifest, target binaryfetch.AcquisitionTarget) (*binaryfetch.BearerAuthorization, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if target.Auth == nil {
		return nil, nil
	}
	if err := target.Auth.ValidateURL(target.URL); err != nil {
		return nil, err
	}
	if err := manifestpkg.ValidateAcquisitionAuthReference(manifest, *target.Auth); err != nil {
		return nil, err
	}
	identity, err := credentialauthority.ParseIdentity(target.Auth.LogicalID)
	if err != nil {
		return nil, errors.New("acquisition credential identity is invalid")
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, errors.New("acquisition credential authority unavailable; use vrooli credentials doctor")
	}
	token, err := authority.Resolve(identity, target.Auth.Field)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return nil, fmt.Errorf("acquisition credential unconfigured; use vrooli credentials provision --identity %s --field %s", target.Auth.LogicalID, target.Auth.Field)
		}
		return nil, errors.New("acquisition credential unavailable; use vrooli credentials doctor")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return binaryfetch.NewBearerAuthorization(token), nil
}
