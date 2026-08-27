package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	portabilityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability/portability_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	mainParameterA = 2
)

type ledger struct {
	Capabilities []capability `json:"capabilities"`
	Resources    []resource   `json:"resources"`
	SkipBudget   skipBudget   `json:"skipBudget"`
}

type capability struct {
	Capability string     `json:"capability"`
	Platforms  []platform `json:"platforms"`
}

type platform struct {
	HostOS        string `json:"hostOs"`
	Qualification string `json:"qualification"`
}

type resource struct {
	Name            string           `json:"name"`
	Driver          string           `json:"driver"`
	AcquisitionKind string           `json:"acquisitionKind"`
	Platforms       []resourceTarget `json:"platforms"`
}

type resourceTarget struct {
	HostOS        string                 `json:"hostOs"`
	Support       string                 `json:"support"`
	Architectures []resourceArchitecture `json:"architectures"`
	Mismatch      bool                   `json:"mismatch"`
	Reason        string                 `json:"reason"`
}

type resourceArchitecture struct {
	Architecture string `json:"architecture"`
	Support      string `json:"support"`
	Reason       string `json:"reason"`
}

type skipBudget struct {
	Available           bool           `json:"available"`
	Measured            int            `json:"measured"`
	Budgets             map[string]int `json:"budgets"`
	Reason              string         `json:"reason"`
	RatchetDirection    string         `json:"ratchetDirection"`
	LastRunWithinBudget bool           `json:"lastRunWithinBudget"`
}

func main() {
	check := len(os.Args) == mainParameterA && os.Args[1] == "--check"
	if len(os.Args) > 1 && !check {
		fmt.Fprintln(os.Stderr, "usage: platform-support [--check]")
		os.Exit(mainParameterA)
	}
	root, err := repositoryRoot()
	if err != nil {
		fatal(err)
	}
	data, err := fetchLedger(context.Background())
	if err != nil {
		fatal(err)
	}
	var snapshot ledger
	if err := json.Unmarshal(data, &snapshot); err != nil {
		fatal(fmt.Errorf("decode capability ledger: %w", err))
	}
	output := filepath.Join(root, "docs", "reference", "platform-support.md")
	existing, err := os.ReadFile(output)
	if err != nil {
		fatal(err)
	}
	generated, err := render(snapshot, existing)
	if err != nil {
		fatal(err)
	}
	if check {
		if !bytes.Equal(existing, generated) {
			fatal(fmt.Errorf("%s is stale; run platform-support generator", output))
		}
		return
	}
	if err := os.WriteFile(output, generated, tuning.PermFile); err != nil {
		fatal(err)
	}
}

func fetchLedger(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, tuning.StandardOperationTimeout)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, "infrastructure-manager")
	if err != nil {
		return nil, fmt.Errorf("read capability ledger: resolve infrastructure-manager: %w", err)
	}
	client := portabilityconnect.NewPortabilityServiceClient(&http.Client{Timeout: tuning.StandardOperationTimeout}, base)
	response, err := client.GetGrid(ctx, connect.NewRequest(&portabilityv1.GetGridRequest{}))
	if err != nil {
		return nil, fmt.Errorf("read capability ledger: get grid: %w", err)
	}
	grid := response.Msg.GetGrid()
	if grid == nil {
		return nil, fmt.Errorf("read capability ledger: infrastructure-manager returned no grid")
	}
	// The Connect response is a protobuf message. Use protojson here so enum
	// fields remain their stable symbolic names; encoding/json serializes the
	// generated int32 enum fields as numbers, which would make this typed
	// renderer disagree with the public ledger JSON contract.
	data, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(grid)
	if err != nil {
		return nil, fmt.Errorf("encode capability ledger: %w", err)
	}
	return data, nil
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, repocontractmeta.ProjectConfigDir, "capability-vocabulary.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found")
		}
		dir = parent
	}
}

func render(snapshot ledger, existing []byte) ([]byte, error) {
	marker := "<!-- END GENERATED LEDGER -->"
	index := bytes.Index(existing, []byte(marker))
	if index < 0 {
		return nil, fmt.Errorf("platform support document has no generated-ledger boundary")
	}
	body := existing[index+len(marker):]
	var out strings.Builder
	out.WriteString("<!-- BEGIN GENERATED LEDGER -->\n# Platform Support and Evidence Matrix\n\n")
	out.WriteString("> Generated by `go run ./internal/deployability/cmd/platform-support` from `vrooli capability ledger --json`. The ledger is the computed source; this document is a rendered readout.\n\n")
	out.WriteString("## Ledger snapshot\n\n| Host OS | Qualified | Build verified | Unqualified | Undeclared | Ineligible |\n|---|---:|---:|---:|---:|---:|\n")
	counts := map[string]map[string]int{}
	for _, item := range snapshot.Capabilities {
		for _, cell := range item.Platforms {
			if _, ok := counts[cell.HostOS]; !ok {
				counts[cell.HostOS] = map[string]int{}
			}
			counts[cell.HostOS][cell.Qualification]++
		}
	}
	for _, hostOS := range []string{"HOST_OS_LINUX", "HOST_OS_MACOS", "HOST_OS_WINDOWS"} {
		label := strings.ToLower(strings.TrimPrefix(hostOS, "HOST_OS_"))
		out.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d |\n", label, counts[hostOS]["QUALIFICATION_QUALIFIED"], counts[hostOS]["QUALIFICATION_BUILD_VERIFIED"], counts[hostOS]["QUALIFICATION_UNQUALIFIED"], counts[hostOS]["QUALIFICATION_UNDECLARED"], counts[hostOS]["QUALIFICATION_INELIGIBLE"]))
	}
	out.WriteString("\n## Resource architecture claims\n\n| Resource | Driver | Acquisition | Host OS | Architecture cells | OS support | Mismatch | Reason |\n|---|---|---|---|---|---|---|---|\n")
	resources := append([]resource(nil), snapshot.Resources...)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})
	if len(resources) == 0 {
		out.WriteString("| _unavailable_ | — | — | — | — | — | — | resource claim source returned no entries |\n")
	} else {
		for _, item := range resources {
			for _, platform := range item.Platforms {
				architectures := make([]string, 0, len(platform.Architectures))
				for _, architecture := range platform.Architectures {
					architectures = append(architectures, architecture.Architecture+"="+architecture.Support)
				}
				out.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %t | %s |\n", item.Name, item.Driver, item.AcquisitionKind, platform.HostOS, strings.Join(architectures, ", "), platform.Support, platform.Mismatch, firstNonEmpty(platform.Reason, "—")))
			}
		}
	}
	out.WriteString("\n## Platform skip budget\n\n")
	if !snapshot.SkipBudget.Available {
		out.WriteString("**Unavailable:** " + snapshot.SkipBudget.Reason + "\n")
	} else {
		out.WriteString(fmt.Sprintf("Measured platform-gated skips: **%d**. Per-OS budgets: `%v`. Ratchet: **%s**. Last run within budget: **%t**.\n", snapshot.SkipBudget.Measured, snapshot.SkipBudget.Budgets, firstNonEmpty(snapshot.SkipBudget.RatchetDirection, "unspecified"), snapshot.SkipBudget.LastRunWithinBudget))
	}
	out.WriteString(marker)
	out.Write(body)
	return []byte(out.String()), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
