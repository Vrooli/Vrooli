package physicalsetups

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const GroupName = "physical-setups"
const maxSetupBytes = 256 << 10

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	hc, base := cliapp.NewConnectHTTPClient(core)
	client := api.NewShotsServiceClient(hc, base)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ShotsService.SavePhysicalShotPreset": cliapp.ProtoMutation(func(op cliapp.OperationContext) (anyMessage, error) {
			setup, err := readSetup(op.Flag("setup"))
			if err != nil {
				return nil, err
			}
			expected, err := parseUint32(op.Flag("expected-revision"), true)
			if err != nil {
				return nil, err
			}
			r, err := client.SavePhysicalShotPreset(context.Background(), connect.NewRequest(&v1.SavePhysicalShotPresetRequest{PresetId: op.Positional("preset-id"), ExpectedRevision: expected, Name: op.Flag("name"), Setup: setup}))
			if err != nil {
				return nil, fmt.Errorf("save physical setup: %w", err)
			}
			if r.Msg.Revision == nil {
				return nil, fmt.Errorf("server returned no saved physical setup revision")
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyMessage) cliapp.MutationReport {
			r := response.(*v1.SavePhysicalShotPresetResponse).Revision
			return cliapp.MutationReport{Result: []string{describe(r)}, Changes: []string{describe(r)}, NextCommand: []string{fmt.Sprintf("`physical-setups get %s --revision %d` — inspect the saved immutable setup", r.PresetId, r.Revision)}}
		}),
		"ShotsService.GetPhysicalShotPreset": cliapp.ProtoList(func(op cliapp.OperationContext) (anyMessage, error) {
			revision, err := parseUint32(op.Flag("revision"), false)
			if err != nil {
				return nil, err
			}
			r, err := client.GetPhysicalShotPreset(context.Background(), connect.NewRequest(&v1.GetPhysicalShotPresetRequest{PresetId: op.Positional("preset-id"), Revision: revision}))
			if err != nil {
				return nil, fmt.Errorf("get physical setup: %w", err)
			}
			if r.Msg.Revision == nil {
				return nil, fmt.Errorf("server returned no physical setup revision")
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyMessage) cliapp.ListReport {
			return report(response.(*v1.GetPhysicalShotPresetResponse).Revision, "Loaded immutable physical setup revision.")
		}),
		"ShotsService.GetPhysicalShotPresetHead": cliapp.ProtoList(func(op cliapp.OperationContext) (anyMessage, error) {
			r, err := client.GetPhysicalShotPresetHead(context.Background(), connect.NewRequest(&v1.GetPhysicalShotPresetHeadRequest{PresetId: op.Positional("preset-id")}))
			if err != nil {
				return nil, fmt.Errorf("get physical setup head: %w", err)
			}
			if r.Msg.Revision == nil {
				return nil, fmt.Errorf("server returned no physical setup head")
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyMessage) cliapp.ListReport {
			return report(response.(*v1.GetPhysicalShotPresetHeadResponse).Revision, "Current immutable physical setup head; qualification remains separate.")
		}),
		"ShotsService.ListPhysicalShotPresets": cliapp.ProtoList(func(op cliapp.OperationContext) (anyMessage, error) {
			size, err := parsePage(op.Flag("page-size"))
			if err != nil {
				return nil, err
			}
			after := op.Flag("after-preset-id")
			if len(after) > 128 || strings.ContainsAny(after, "\\/\x00\r\n") {
				return nil, fmt.Errorf("after-preset-id must be a stable cursor")
			}
			r, err := client.ListPhysicalShotPresets(context.Background(), connect.NewRequest(&v1.ListPhysicalShotPresetsRequest{AfterPresetId: after, PageSize: size}))
			if err != nil {
				return nil, fmt.Errorf("list physical setups: %w", err)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyMessage) cliapp.ListReport {
			r := response.(*v1.ListPhysicalShotPresetsResponse)
			rows := make([]string, 0, len(r.Presets))
			for _, row := range r.Presets {
				rows = append(rows, describe(row))
			}
			hints := []string{}
			if r.NextAfterPresetId != "" {
				hints = append(hints, fmt.Sprintf("`physical-setups list --after-preset-id %s` — next page", r.NextAfterPresetId))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d immutable candidate setup(s).", len(rows))}, ResultsHeading: "Physical setup revisions", Results: rows, RetrievalHints: hints}
		}),
		"ShotsService.ListPhysicalShotPresetRevisions": cliapp.ProtoList(func(op cliapp.OperationContext) (anyMessage, error) {
			after, err := parseOptionalUint32(op.Flag("after-revision"))
			if err != nil {
				return nil, err
			}
			size, err := parsePage(op.Flag("page-size"))
			if err != nil {
				return nil, err
			}
			r, err := client.ListPhysicalShotPresetRevisions(context.Background(), connect.NewRequest(&v1.ListPhysicalShotPresetRevisionsRequest{PresetId: op.Positional("preset-id"), AfterRevision: after, PageSize: size}))
			if err != nil {
				return nil, fmt.Errorf("list physical setup revisions: %w", err)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, response anyMessage) cliapp.ListReport {
			r := response.(*v1.ListPhysicalShotPresetRevisionsResponse)
			rows := make([]string, 0, len(r.Revisions))
			for _, row := range r.Revisions {
				rows = append(rows, describe(row))
			}
			hints := []string{}
			if r.NextAfterRevision > 0 {
				hints = append(hints, fmt.Sprintf("`physical-setups revisions %s --after-revision %d` — next page", r.Revisions[0].PresetId, r.NextAfterRevision))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d immutable setup revision(s).", len(rows))}, ResultsHeading: "Physical setup revisions", Results: rows, RetrievalHints: hints}
		}),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("physical-setups: load manifest: %w", err)
	}
	return group, nil
}

type anyMessage = proto.Message

func readSetup(path string) (*v1.PhysicalShotSetup, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("setup is required; pass a strict PhysicalShotSetup proto-JSON file path or - for stdin")
	}
	var reader io.Reader
	var file *os.File
	if path == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open setup: %w", err)
		}
		file, reader = f, f
		defer file.Close()
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxSetupBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read setup: %w", err)
	}
	if len(data) == 0 || len(data) > maxSetupBytes {
		return nil, fmt.Errorf("setup must be nonempty and at most %d bytes", maxSetupBytes)
	}
	setup := &v1.PhysicalShotSetup{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, setup); err != nil {
		return nil, fmt.Errorf("decode strict PhysicalShotSetup proto-JSON: %w", err)
	}
	return setup, nil
}
func parseUint32(raw string, zeroOK bool) (uint32, error) {
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || (!zeroOK && n == 0) {
		return 0, fmt.Errorf("revision must be a valid %s 32-bit integer", map[bool]string{true: "nonnegative", false: "positive"}[zeroOK])
	}
	return uint32(n), nil
}
func parsePage(raw string) (uint32, error) {
	if raw == "" {
		return 50, nil
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n < 1 || n > 100 {
		return 0, fmt.Errorf("page-size must be from 1 to 100")
	}
	return uint32(n), nil
}
func parseOptionalUint32(raw string) (uint32, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return parseUint32(raw, true)
}
func describe(r *v1.PhysicalShotPresetRevision) string {
	if r == nil {
		return "invalid empty preset revision"
	}
	return fmt.Sprintf("preset=%s revision=%d state=%s candidate_only=%t sha256=%s name=%q", r.PresetId, r.Revision, r.State, r.CandidateOnly, r.Sha256, r.Name)
}
func report(r *v1.PhysicalShotPresetRevision, summary string) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Physical setup revision", Results: []string{describe(r)}}
}
