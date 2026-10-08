// Package releasesvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.releases.ReleasesService and the shared
// application logic behind the REST release routes. Building and verifying
// live in package release; this package adapts requests, resolves the store
// and repository, and maps every failure onto the typed apierrors.Error.
package releasesvc

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/release"

	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/releases"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/releases/releasesv1connect"
)

// SchemaVersion is the response schema version clients negotiate on.
const SchemaVersion = "1"

// Config binds the service to a repository and a store.
type Config struct {
	// RepoRoot is the Vrooli repository releases are built from.
	RepoRoot string
	// StoreDir is the bundle store; releases live in StoreDir/releases.
	StoreDir string
	// NativeCLI pins the control-plane build (defaults: ./cmd/vrooli at RepoRoot).
	NativeCLI release.NativeCLIOptions
	// Signer signs production releases; nil refuses production builds.
	Signer release.Signer
	// PublicKeyPath is the trust anchor; empty means RepoRoot/install/vrooli-release.pub.
	PublicKeyPath string
	// TrustMode overrides configuration when set.
	TrustMode release.TrustMode
}

// Service implements releasesv1connect.ReleasesServiceHandler.
type Service struct {
	cfg Config
}

// New builds the service.
func New(cfg Config) *Service { return &Service{cfg: cfg} }

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return releasesv1connect.NewReleasesServiceHandler(s, opts...)
}

// Store returns the release store the service reads.
func (s *Service) Store() release.Store { return release.Store{Dir: s.cfg.StoreDir} }

// BuildRequest is the transport-neutral build request.
type BuildRequest struct {
	Manifest           domain.CloudManifest `json:"manifest"`
	ClosureDigest      string               `json:"closure_digest,omitempty"`
	GOOS               string               `json:"goos"`
	GOARCH             string               `json:"goarch"`
	TrustMode          string               `json:"trust_mode,omitempty"`
	VerifyReproducible bool                 `json:"verify_reproducible,omitempty"`
}

// Build builds (or returns the identical existing) release.
func (s *Service) Build(ctx context.Context, req BuildRequest) (release.Release, error) {
	if strings.TrimSpace(s.cfg.RepoRoot) == "" || strings.TrimSpace(s.cfg.StoreDir) == "" {
		return release.Release{}, apierrors.New(apierrors.CodeInternal, "release service has no repository root or store configured")
	}
	mode := s.cfg.TrustMode
	if strings.TrimSpace(req.TrustMode) != "" {
		mode = release.TrustMode(strings.TrimSpace(req.TrustMode))
	}
	native := s.cfg.NativeCLI
	native.VerifyReproducible = req.VerifyReproducible
	return release.Build(ctx, release.BuildInputs{
		RepoRoot:      s.cfg.RepoRoot,
		StoreDir:      s.cfg.StoreDir,
		Manifest:      req.Manifest,
		ClosureDigest: req.ClosureDigest,
		Platform:      release.Platform{GOOS: req.GOOS, GOARCH: req.GOARCH},
		NativeCLI:     native,
		TrustMode:     mode,
		Signer:        s.cfg.Signer,
		PublicKeyPath: s.cfg.PublicKeyPath,
	})
}

// Get reads one complete release.
func (s *Service) Get(digest string) (release.Release, error) {
	return s.Store().Load(digest)
}

// VerifyRequest is the transport-neutral verify request.
type VerifyRequest struct {
	TrustMode string `json:"trust_mode,omitempty"`
	GOOS      string `json:"goos,omitempty"`
	GOARCH    string `json:"goarch,omitempty"`
}

// Verify runs every cloud-side trust check on a stored release.
func (s *Service) Verify(ctx context.Context, digest string, req VerifyRequest) (release.VerifyReport, error) {
	dir, err := s.Store().Path(digest)
	if err != nil {
		return release.VerifyReport{}, err
	}
	if _, err := s.Store().Load(digest); err != nil {
		typed := apierrors.As(err)
		if typed != nil && typed.Details["reason"] == release.ReasonNotFound {
			return release.VerifyReport{}, err
		}
	}
	opts := release.VerifyOptions{RepoRoot: s.cfg.RepoRoot, PublicKeyPath: s.cfg.PublicKeyPath, TrustMode: s.cfg.TrustMode}
	if strings.TrimSpace(req.TrustMode) != "" {
		opts.TrustMode = release.TrustMode(strings.TrimSpace(req.TrustMode))
	}
	if req.GOOS != "" || req.GOARCH != "" {
		opts.Platform = &release.Platform{GOOS: req.GOOS, GOARCH: req.GOARCH}
	}
	return release.Verify(ctx, dir, opts)
}

// BuildRelease implements the Connect RPC.
func (s *Service) BuildRelease(ctx context.Context, req *connect.Request[releasesv1.BuildReleaseRequest]) (*connect.Response[releasesv1.BuildReleaseResponse], error) {
	var manifest domain.CloudManifest
	if req.Msg.GetManifest() == nil {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeManifestInvalid, "manifest is required"))
	}
	raw, err := req.Msg.GetManifest().MarshalJSON()
	if err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Newf(apierrors.CodeManifestInvalid, "encode manifest: %v", err))
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, deploymentsvc.ConnectError(apierrors.Newf(apierrors.CodeManifestInvalid, "decode manifest: %v", err))
	}
	rel, err := s.Build(ctx, BuildRequest{Manifest: manifest, ClosureDigest: req.Msg.GetClosureDigest(), GOOS: req.Msg.GetGoos(), GOARCH: req.Msg.GetGoarch(), TrustMode: req.Msg.GetTrustMode(), VerifyReproducible: req.Msg.GetVerifyReproducible()})
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(&releasesv1.BuildReleaseResponse{SchemaVersion: SchemaVersion, Release: ReleaseProto(rel)}), nil
}

// GetRelease implements the Connect RPC.
func (s *Service) GetRelease(_ context.Context, req *connect.Request[releasesv1.GetReleaseRequest]) (*connect.Response[releasesv1.GetReleaseResponse], error) {
	rel, err := s.Get(req.Msg.GetReleaseDigest())
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(&releasesv1.GetReleaseResponse{SchemaVersion: SchemaVersion, Release: ReleaseProto(rel)}), nil
}

// VerifyRelease implements the Connect RPC.
func (s *Service) VerifyRelease(ctx context.Context, req *connect.Request[releasesv1.VerifyReleaseRequest]) (*connect.Response[releasesv1.VerifyReleaseResponse], error) {
	report, err := s.Verify(ctx, req.Msg.GetReleaseDigest(), VerifyRequest{TrustMode: req.Msg.GetTrustMode(), GOOS: req.Msg.GetGoos(), GOARCH: req.Msg.GetGoarch()})
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(VerifyProto(report)), nil
}

// ReleaseProto projects a stored release onto the wire message.
func ReleaseProto(rel release.Release) *releasesv1.Release {
	m := rel.Manifest
	in := rel.Inputs
	out := &releasesv1.Release{
		ReleaseDigest: rel.Digest, Dir: rel.Dir, Complete: rel.Complete,
		Manifest: &releasesv1.ReleaseManifest{
			SchemaVersion: int32(m.SchemaVersion), ReleaseDigest: m.ReleaseDigest, BundleSha256: m.BundleSHA256,
			NativeCli:     &releasesv1.NativeCLI{Sha256: m.NativeCLI.SHA256, Goos: m.NativeCLI.GOOS, Goarch: m.NativeCLI.GOARCH},
			ClosureDigest: m.ClosureDigest, ConfigurationDigest: m.ConfigurationDigest,
			Provenance: &releasesv1.Provenance{Builder: m.Provenance.Builder, Policy: m.Provenance.Policy},
			Limits:     &releasesv1.Limits{MaxEntries: m.Limits.MaxEntries, MaxExpandedBytes: m.Limits.MaxExpandedBytes, MaxEntryBytes: m.Limits.MaxEntryBytes},
		},
		Inputs: &releasesv1.ReleaseInputs{
			SchemaVersion: int32(in.SchemaVersion), ReleaseDigest: in.ReleaseDigest, BuiltAt: in.BuiltAt,
			Builder: &releasesv1.ReleaseBuilder{Identity: in.Builder.Identity, Node: in.Builder.Node, User: in.Builder.User},
			Source: &releasesv1.ReleaseSource{
				Commit: in.Source.Commit, Dirty: in.Source.Dirty, DirtyPaths: int32(in.Source.DirtyPaths), Snapshot: in.Source.Snapshot,
				ContentManifestSha256: in.Source.ContentManifestSHA256, FileCount: int32(in.Source.FileCount), IncludeRoots: in.Source.IncludeRoots, Excludes: in.Source.Excludes,
			},
			Bundle: &releasesv1.ReleaseBundle{FileName: in.Bundle.FileName, Sha256: in.Bundle.SHA256, SizeBytes: in.Bundle.SizeBytes},
			Dependencies: &releasesv1.ReleaseDependencies{
				ClosureDigest: in.Dependencies.ClosureDigest, AnalyzerTool: in.Dependencies.AnalyzerTool, AnalyzerFingerprint: in.Dependencies.AnalyzerFingerprint,
				AnalyzerGeneratedAt: in.Dependencies.AnalyzerGeneratedAt, Scenarios: in.Dependencies.Scenarios, Resources: in.Dependencies.Resources,
			},
			NativeCli: &releasesv1.ReleaseNativeCLIBuild{
				FileName: in.NativeCLI.FileName, Sha256: in.NativeCLI.SHA256, Goos: in.NativeCLI.GOOS, Goarch: in.NativeCLI.GOARCH, SizeBytes: in.NativeCLI.SizeBytes,
				Package: in.NativeCLI.Package, ModuleDir: in.NativeCLI.ModuleDir, GoVersion: in.NativeCLI.GoVersion, Args: in.NativeCLI.Args, Env: in.NativeCLI.Env,
			},
			Toolchain:       &releasesv1.ReleaseToolchain{GoVersion: in.Toolchain.GoVersion, HostGoos: in.Toolchain.HostGOOS, HostGoarch: in.Toolchain.HostGOARCH, GoFlags: in.Toolchain.GoFlags},
			Configuration:   &releasesv1.ReleaseConfiguration{Digest: in.Configuration.Digest, Schema: in.Configuration.Schema, Rule: in.Configuration.Rule},
			Provenance:      &releasesv1.ReleaseProvenance{Policy: in.Provenance.Policy, TrustMode: in.Provenance.TrustMode, SignerKeyId: in.Provenance.SignerKeyID, SignatureFile: in.Provenance.SignatureFile},
			Reproducibility: &releasesv1.ReleaseReproducibility{Bundle: in.Reproducibility.Bundle, NativeCli: in.Reproducibility.NativeCLI, Reason: in.Reproducibility.Reason},
			Limitations:     in.Limitations,
		},
	}
	for _, artifact := range in.ResourceArtifacts {
		out.Inputs.ResourceArtifacts = append(out.Inputs.ResourceArtifacts, &releasesv1.ReleaseResourceArtifact{
			Component: artifact.Component, Platform: artifact.Platform, Name: artifact.Name, Digest: artifact.Digest, Mode: artifact.Mode, Eligibility: artifact.Eligibility, LicenseRefs: artifact.LicenseRefs,
		})
	}
	for _, credential := range in.Credentials {
		ref := &releasesv1.ReleaseCredentialRef{Id: credential.ID, Class: credential.Class, Required: credential.Required, TargetType: credential.TargetType, TargetName: credential.TargetName}
		if credential.Descriptor != nil {
			ref.LogicalId = credential.Descriptor.LogicalID
			ref.Field = credential.Descriptor.Field
		}
		out.Inputs.Credentials = append(out.Inputs.Credentials, ref)
	}
	return out
}

// VerifyProto projects a verification report onto the wire message.
func VerifyProto(report release.VerifyReport) *releasesv1.VerifyReleaseResponse {
	out := &releasesv1.VerifyReleaseResponse{SchemaVersion: SchemaVersion, ReleaseDigest: report.ReleaseDigest, Verified: report.Verified, Policy: report.Policy, TrustMode: report.TrustMode, SignerKeyId: report.SignerKeyID}
	for _, check := range report.Checks {
		out.Checks = append(out.Checks, &releasesv1.VerifyCheck{Id: check.ID, Status: check.Status, Code: check.Code})
	}
	return out
}

// ManifestStruct encodes a manifest for a Connect build request.
func ManifestStruct(m domain.CloudManifest) (*structpb.Struct, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	return structpb.NewStruct(generic)
}
