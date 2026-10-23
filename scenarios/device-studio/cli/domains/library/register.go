package library

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	catalog "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	physical "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	physicalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical/physical_v1connect"
	"google.golang.org/protobuf/proto"
)

const GroupName = "library"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	client := physicalconnect.NewPhysicalStarterServiceClient(hc, base)
	catalogClient := catalogconnect.NewLibraryCatalogServiceClient(hc, base)
	intake := newIntakeHandlers(core)
	call := func(_ cliapp.OperationContext) (*physical.InitializeReviewedStarterResponse, error) {
		r, e := client.InitializeReviewedStarter(context.Background(), connect.NewRequest(&physical.InitializeReviewedStarterRequest{}))
		if e != nil {
			return nil, cliapp.WrapAPIError("initialize reviewed starter", e, nil)
		}
		if r.Msg.PhoneDevice == nil || r.Msg.PhoneStudyScene == nil || r.Msg.PhoneStillMotion == nil || r.Msg.PhoneStudyPlacement == nil {
			return nil, fmt.Errorf("starter service returned incomplete candidate references")
		}
		if !r.Msg.CandidateOnly {
			return nil, fmt.Errorf("starter service returned a non-candidate starter state")
		}
		return r.Msg, nil
	}
	report := func(_ cliapp.OperationContext, r *physical.InitializeReviewedStarterResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Reviewed starter is ready: device %s@%d, scene %s@%d, motion %s@%d.", r.PhoneDevice.Id, r.PhoneDevice.Revision, r.PhoneStudyScene.Id, r.PhoneStudyScene.Revision, r.PhoneStillMotion.Id, r.PhoneStillMotion.Revision), "These are reviewed candidate revisions; render qualification is a separate decision."}, Changes: []string{fmt.Sprintf("candidate_only=%t reviewed_artifacts=%d placement=%s", r.CandidateOnly, len(r.ReviewedArtifacts), r.PhoneStudyPlacement.ProfileId)}, NextCommand: []string{fmt.Sprintf("`devices head %s` — inspect the starter device", r.PhoneDevice.Id), fmt.Sprintf("`scenes head %s` — inspect the starter scene", r.PhoneStudyScene.Id), fmt.Sprintf("`motions head %s` — inspect the starter motion", r.PhoneStillMotion.Id)}}
	}
	bindings := map[string]cliapp.PrimitiveHandler{
		"PhysicalStarterService.InitializeReviewedStarter": cliapp.ProtoMutation(call, report),
		"LibraryCatalogService.ListBundleRevisions": cliapp.ProtoList(func(op cliapp.OperationContext) (anyProtoMessage, error) {
			pageSize, err := boundedPageSize(op.Flag("page-size"))
			if err != nil {
				return nil, err
			}
			afterID := op.Flag("after-bundle-id")
			if afterID != "" && !validStableID(afterID) {
				return nil, fmt.Errorf("after-bundle-id must be a stable identifier of at most 128 characters")
			}
			afterRevision, err := parseRevision(op.Flag("after-revision"))
			if err != nil {
				return nil, err
			}
			if afterID == "" && afterRevision != 0 {
				return nil, fmt.Errorf("after-revision requires after-bundle-id")
			}
			r, err := catalogClient.ListBundleRevisions(context.Background(), connect.NewRequest(&catalog.ListBundleRevisionsRequest{AfterBundleId: afterID, AfterRevision: afterRevision, PageSize: pageSize}))
			if err != nil {
				return nil, cliapp.WrapAPIError("list library bundle revisions", err, nil)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyProtoMessage) cliapp.ListReport {
			r := response.(*catalog.ListBundleRevisionsResponse)
			results := make([]string, 0, len(r.Revisions))
			for _, revision := range r.Revisions {
				results = append(results, describeBundle(revision))
			}
			hints := []string{}
			if r.NextAfterBundleId != "" {
				hints = append(hints, fmt.Sprintf("`library list --after-bundle-id %s --after-revision %d` — next page", r.NextAfterBundleId, r.NextAfterRevision))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d reviewed bundle revision(s); asset bytes are not included.", len(results))}, ResultsHeading: "Library bundle revisions", Results: results, RetrievalHints: hints}
		}),
		"LibraryCatalogService.GetBundleRevision": cliapp.ProtoList(func(op cliapp.OperationContext) (anyProtoMessage, error) {
			revision, err := parseRequiredRevision(op.Flag("revision"))
			if err != nil {
				return nil, err
			}
			r, err := catalogClient.GetBundleRevision(context.Background(), connect.NewRequest(&catalog.GetBundleRevisionRequest{BundleId: op.Positional("bundle-id"), Revision: revision}))
			if err != nil {
				return nil, cliapp.WrapAPIError("get library bundle revision", err, nil)
			}
			if r.Msg.Revision == nil {
				return nil, fmt.Errorf("server returned no bundle revision metadata")
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyProtoMessage) cliapp.ListReport {
			r := response.(*catalog.GetBundleRevisionResponse)
			return cliapp.ListReport{Summary: []string{"Loaded immutable reviewed bundle metadata. Artifact storage details and bytes are not exposed."}, ResultsHeading: "Library bundle revision", Results: []string{describeBundle(r.Revision)}}
		}),
		"LibraryCatalogService.ListIntakeCandidates": cliapp.ProtoList(intake.listCall, intake.listReport),
		"LibraryCatalogService.GetIntakeCandidate":   cliapp.ProtoList(intake.getCall, intake.getReport),
	}
	g, e := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("library: load manifest: %w", e)
	}
	g.Subcommands = append(g.Subcommands, cliapp.Command{
		Name:        "intake-import",
		Description: "Copy a local GLB into geometry intake and record unverified source claims",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "file", Required: true, Description: "Path to the local GLB file"},
			{Name: "candidate-id", Required: true, Description: "Stable idempotency key for this import"},
			{Name: "display-name", Required: true, Description: "Human-readable candidate name"},
			{Name: "source-label", Required: true, Description: "How this local file was obtained"},
			{Name: "license-claim", Description: "Operator-supplied license claim; not a verified grant"},
			{Name: "attribution", Description: "Operator-supplied attribution claim"},
			{Name: "declared-uses", Description: "Comma-separated intended-use claims (not granted rights)"},
			{Name: "parent-id", Description: "Parent candidate ID for a derived revision"},
			{Name: "parent-revision", Description: "Exact parent candidate revision"},
			{Name: "parent-sha256", Description: "Exact parent candidate revision digest"},
		}},
		Architecture: cliapp.CommandArchitecture{Exception: cliapp.ExceptionUpload, ExceptionReason: "REST multipart local GLB intake; original bytes are streamed and digest verified"},
	}.WithPrimitive(cliapp.Upload(intake.importCall, intake.importReport)))
	g.Subcommands = append(g.Subcommands,
		cliapp.Command{Name: "intake-materials", Description: "Inspect material factors from an exact imported GLB revision", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Imported candidate ID"}}, Flags: []cliapp.Flag{{Name: "revision", Required: true, Description: "Exact immutable revision"}, {Name: "revision-sha256", Required: true, Description: "Exact candidate revision digest"}}}}.WithPrimitive(cliapp.Action(intake.materialsCall, intake.materialsReport)),
		cliapp.Command{Name: "intake-preview", Description: "Write a bounded read-only vertex-color preview of an exact unreviewed GLB revision", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Imported candidate ID"}}, Flags: []cliapp.Flag{{Name: "revision", Required: true, Description: "Exact immutable revision"}, {Name: "revision-sha256", Required: true, Description: "Exact candidate revision digest"}, {Name: "out", Required: true, Description: "New PNG output path"}}}}.WithPrimitive(cliapp.Action(intake.previewCall, intake.previewReport)),
		cliapp.Command{Name: "intake-material-correct", Description: "Create an unreviewed child with deterministic local PBR material edits", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "candidate-id", Required: true, Description: "Imported parent candidate ID"}}, Flags: []cliapp.Flag{{Name: "revision", Required: true, Description: "Exact parent revision"}, {Name: "revision-sha256", Required: true, Description: "Exact parent candidate revision digest"}, {Name: "child-candidate-id", Required: true, Description: "Stable child operation identity; reuse on uncertain retry"}, {Name: "material-index", Required: true, Description: "Index from intake-materials"}, {Name: "material-name", Required: true, Description: "Exact observed material name"}, {Name: "base-color", Description: "Optional RGBA factors, four comma-separated values in [0,1]"}, {Name: "metallic", Description: "Optional metallic factor in [0,1]"}, {Name: "roughness", Description: "Optional roughness factor in [0,1]"}, {Name: "emissive", Description: "Optional RGB factors, three comma-separated values in [0,1]"}}}}.WithPrimitive(cliapp.Action(intake.correctMaterialCall, intake.correctionReport)),
	)
	return g, nil
}

type anyProtoMessage = proto.Message

func describeBundle(revision *catalog.BundleRevisionMetadata) string {
	if revision == nil || revision.Manifest == nil || revision.Review == nil {
		return "invalid bundle metadata response"
	}
	artifactFacts := make([]string, 0, len(revision.Manifest.Artifacts))
	for _, artifact := range revision.Manifest.Artifacts {
		if artifact != nil {
			artifactFacts = append(artifactFacts, fmt.Sprintf("%s:%s:%s rights=%v", artifact.Id, artifact.Role, artifact.LicenseId, artifact.Rights))
		}
	}
	return fmt.Sprintf("%s@%d manifest=%s review=%s decision=%s allowed_uses=%v artifacts=[%s]", revision.Manifest.BundleId, revision.Manifest.Revision, revision.ManifestSha256, revision.ReviewSha256, revision.Review.Decision, revision.Review.AllowedUses, strings.Join(artifactFacts, "; "))
}

func validStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 || !alphaNumeric(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		ch := value[i]
		if !alphaNumeric(ch) && ch != '.' && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func alphaNumeric(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func boundedPageSize(raw string) (uint32, error) {
	value, err := parseRevision(raw)
	if err != nil {
		return 0, fmt.Errorf("page-size must be a number between 1 and 100")
	}
	if value == 0 {
		return 50, nil
	}
	if value > 100 {
		return 0, fmt.Errorf("page-size must not exceed 100")
	}
	return value, nil
}

func parseRevision(raw string) (uint32, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("revision must be a non-negative 32-bit integer")
	}
	return uint32(value), nil
}

func parseRequiredRevision(raw string) (uint32, error) {
	value, err := parseRevision(raw)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("revision must be a positive 32-bit integer")
	}
	return value, nil
}
