package ontology

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

type topologyJSON struct {
	Sectors      []topologySector     `json:"sectors"`
	Stages       []topologyStage      `json:"stages"`
	Dependencies []topologyDependency `json:"dependencies"`
}

type topologySector struct {
	Slug      string `json:"slug"`
	ClusterID string `json:"cluster_id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
}

type topologyStage struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	ClusterID string `json:"cluster_id"`
	StageType string `json:"stage_type"`
	Order     int32  `json:"order"`
}

type topologyDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

func ParseTopology(data []byte) (TopologyImport, error) {
	var raw topologyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return TopologyImport{}, fmt.Errorf("decode topology: %w", err)
	}
	return MapTopology(raw)
}

func MapTopology(raw topologyJSON) (TopologyImport, error) {
	out := TopologyImport{
		Capabilities: make([]Capability, 0, len(raw.Sectors)+len(raw.Stages)),
		Edges:        make([]CapabilityEdge, 0, len(raw.Dependencies)),
	}
	sectorIDs := map[string]string{}
	capabilityIDs := map[string]struct{}{}
	for _, sector := range raw.Sectors {
		id, err := NormalizeID("sector.cluster_id", sector.ClusterID)
		if err != nil {
			return TopologyImport{}, err
		}
		slug, err := NormalizeID("sector.slug", sector.Slug)
		if err != nil {
			return TopologyImport{}, err
		}
		name := strings.TrimSpace(sector.Name)
		if name == "" {
			name = DefaultName(slug)
		}
		out.Capabilities = append(out.Capabilities, Capability{
			ID:          id,
			Slug:        slug,
			Name:        name,
			Description: strings.TrimSpace(sector.Category),
			Kind:        ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR,
			Importance:  1,
		})
		sectorIDs[id] = id
		capabilityIDs[id] = struct{}{}
	}
	for _, stage := range raw.Stages {
		id, err := NormalizeID("stage.slug", stage.Slug)
		if err != nil {
			return TopologyImport{}, err
		}
		parentID, err := NormalizeID("stage.cluster_id", stage.ClusterID)
		if err != nil {
			return TopologyImport{}, err
		}
		if _, ok := sectorIDs[parentID]; !ok {
			return TopologyImport{}, ErrInvalidArgument{Field: "stage.cluster_id", Reason: "must reference a sector"}
		}
		kind, err := stageKind(stage.StageType)
		if err != nil {
			return TopologyImport{}, err
		}
		name := strings.TrimSpace(stage.Name)
		if name == "" {
			name = DefaultName(id)
		}
		out.Capabilities = append(out.Capabilities, Capability{
			ID:         id,
			Slug:       id,
			Name:       name,
			Kind:       kind,
			ParentID:   parentID,
			SortOrder:  stage.Order,
			Importance: 1,
		})
		capabilityIDs[id] = struct{}{}
	}
	for _, dependency := range raw.Dependencies {
		from, err := NormalizeID("dependency.from", dependency.From)
		if err != nil {
			return TopologyImport{}, err
		}
		to, err := NormalizeID("dependency.to", dependency.To)
		if err != nil {
			return TopologyImport{}, err
		}
		if _, ok := capabilityIDs[from]; !ok {
			return TopologyImport{}, ErrInvalidArgument{Field: "dependency.from", Reason: "must reference a capability"}
		}
		if _, ok := capabilityIDs[to]; !ok {
			return TopologyImport{}, ErrInvalidArgument{Field: "dependency.to", Reason: "must reference a capability"}
		}
		edgeType, err := dependencyType(dependency.Type)
		if err != nil {
			return TopologyImport{}, err
		}
		out.Edges = append(out.Edges, CapabilityEdge{FromID: from, ToID: to, Type: edgeType})
	}
	sort.SliceStable(out.Capabilities, func(i, j int) bool {
		if out.Capabilities[i].Kind != out.Capabilities[j].Kind {
			return out.Capabilities[i].Kind < out.Capabilities[j].Kind
		}
		if out.Capabilities[i].ParentID != out.Capabilities[j].ParentID {
			return out.Capabilities[i].ParentID < out.Capabilities[j].ParentID
		}
		if out.Capabilities[i].SortOrder != out.Capabilities[j].SortOrder {
			return out.Capabilities[i].SortOrder < out.Capabilities[j].SortOrder
		}
		return out.Capabilities[i].ID < out.Capabilities[j].ID
	})
	return out, nil
}

func stageKind(value string) (ontologyv1.CapabilityKind, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "macro":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY, nil
	case "macro_component":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_COMPONENT, nil
	case "capstone":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPSTONE, nil
	case "simulation":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_SIMULATION, nil
	default:
		return 0, ErrInvalidArgument{Field: "stage.stage_type", Reason: "must be macro, macro_component, capstone, or simulation"}
	}
}

func dependencyType(value string) (ontologyv1.CapabilityEdgeType, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "progression":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION, nil
	case "decomposes":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES, nil
	case "requires":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_REQUIRES, nil
	default:
		return 0, ErrInvalidArgument{Field: "dependency.type", Reason: "must be progression, decomposes, or requires"}
	}
}
