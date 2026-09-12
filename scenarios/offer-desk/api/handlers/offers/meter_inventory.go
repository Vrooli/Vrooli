package offers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"connectrpc.com/connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

type committedMeterInventory struct {
	Source string `json:"source"`
	Meters []struct {
		LimitKey   string   `json:"limit_key"`
		Class      string   `json:"class"`
		DeclaredBy []string `json:"declared_by"`
		BundleKeys []string `json:"bundle_keys"`
		Byok       bool     `json:"byok"`
	} `json:"meters"`
}

type monetizationManifest struct {
	Meters []struct {
		LimitKey string `json:"limit_key"`
	} `json:"meters"`
}

func (s *Service) GetMeterInventory(ctx context.Context, _ *connect.Request[offerspb.MeterInventoryRequest]) (*connect.Response[offerspb.MeterInventoryResponse], error) {
	root, err := findRepoRoot()
	if err != nil {
		return nil, internal(err)
	}
	inventoryPath := filepath.Join(root, "packages", "monetization-go", "meter-inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		return nil, internal(fmt.Errorf("read meter inventory: %w", err))
	}
	var committed committedMeterInventory
	if err := json.Unmarshal(data, &committed); err != nil {
		return nil, internal(fmt.Errorf("parse meter inventory: %w", err))
	}
	declaredByScenario := map[string]map[string]bool{}
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "monetization.json"))
	if err != nil {
		return nil, internal(err)
	}
	for _, path := range paths {
		var manifest monetizationManifest
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, internal(err)
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, internal(fmt.Errorf("parse %s: %w", path, err))
		}
		scenario := filepath.Base(filepath.Dir(filepath.Dir(path)))
		keys := declaredByScenario[scenario]
		if keys == nil {
			keys = map[string]bool{}
			declaredByScenario[scenario] = keys
		}
		for _, meter := range manifest.Meters {
			keys[meter.LimitKey] = true
		}
	}

	response := &offerspb.MeterInventoryResponse{Source: committed.Source}
	known := map[string]bool{}
	for _, meter := range committed.Meters {
		known[meter.LimitKey] = true
		response.Meters = append(response.Meters, &offerspb.MeterInventoryEntry{LimitKey: meter.LimitKey, Class: meter.Class, DeclaredBy: append([]string{}, meter.DeclaredBy...), BundleKeys: append([]string{}, meter.BundleKeys...), Byok: meter.Byok})
	}
	nodes, err := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, internal(err)
	}
	byID := make(map[string]*offerspb.Node, len(nodes))
	for _, node := range nodes {
		byID[node.Id] = node
		if node.Kind == offerspb.NodeKind_STREAM && !known[node.Name] {
			response.UndeclaredStreams = append(response.UndeclaredStreams, node.Name)
		}
	}
	edges, err := s.store.ListEdges(ctx, "")
	if err != nil {
		return nil, internal(err)
	}
	for _, edge := range edges {
		if edge.Kind != "unlocks" {
			continue
		}
		from, stream := byID[edge.FromId], byID[edge.ToId]
		if from == nil || stream == nil || from.Kind != offerspb.NodeKind_DELIVERABLE || stream.Kind != offerspb.NodeKind_STREAM {
			continue
		}
		if !declaredByScenario[from.Name][stream.Name] {
			response.DeliverableMeterGaps = append(response.DeliverableMeterGaps, from.Name+" -> "+stream.Name)
		}
	}
	sort.Strings(response.UndeclaredStreams)
	sort.Strings(response.DeliverableMeterGaps)
	return connect.NewResponse(response), nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "scenarios")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "packages")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s", dir)
		}
		dir = parent
	}
}
