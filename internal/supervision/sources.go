package supervision

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/resources"
)

type ManagedServiceSource struct {
	read func() ([]resources.ManagedServiceRecord, error)
}

func NewManagedServiceSource() ManagedServiceSource {
	return ManagedServiceSource{read: resources.ReadManagedServiceRecords}
}

func (s ManagedServiceSource) Owners() ([]Owner, error) {
	read := s.read
	if read == nil {
		read = resources.ReadManagedServiceRecords
	}
	records, err := read()
	if err != nil {
		return nil, err
	}
	owners := make([]Owner, 0, len(records))
	for _, record := range records {
		owners = append(owners, Owner{
			Kind:         OwnerKindResource,
			Name:         record.Resource,
			PID:          record.State.PID,
			ArtifactPath: record.State.ArtifactPath,
			StartedAt:    record.State.StartedAt,
		})
	}
	return owners, nil
}

type ScenarioSource struct {
	home        string
	readRecords func(string, string) ([]process.Record, error)
}

type ResourceProcessSource struct {
	home string
	read func(string) ([]resources.ResourceProcessRecord, error)
}

func NewResourceProcessSource(home string) ResourceProcessSource {
	return ResourceProcessSource{home: home, read: resources.ReadResourceProcessRecords}
}

func (s ResourceProcessSource) Owners() ([]Owner, error) {
	read := s.read
	if read == nil {
		read = resources.ReadResourceProcessRecords
	}
	records, err := read(s.home)
	if err != nil {
		return nil, err
	}
	owners := make([]Owner, 0, len(records))
	for _, record := range records {
		owners = append(owners, Owner{
			Kind:      OwnerKindResource,
			Name:      record.Resource + "/" + record.Name,
			PID:       record.PID,
			StartedAt: record.StartedAt,
		})
	}
	return owners, nil
}

func NewScenarioSource(home string) ScenarioSource {
	return ScenarioSource{home: home, readRecords: process.ReadScenarioRecords}
}

// BuildHostIndex composes the two durable ownership authorities used by the
// control plane. It remains database-free and accepts the runtime home needed
// by scenario process records.
func BuildHostIndex(home string) (*Index, error) {
	return BuildIndex(
		NativeProcessTableSource{},
		hostOwnershipSources(home)...,
	)
}

func hostOwnershipSources(home string) []OwnershipSource {
	return []OwnershipSource{
		NewManagedServiceSource(),
		NewResourceProcessSource(home),
		NewScenarioSource(home),
	}
}

func (s ScenarioSource) Owners() ([]Owner, error) {
	processesDir, err := repocontract.RuntimeHomeEntryPath(s.home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(processesDir, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Owner{}, nil
		}
		return nil, fmt.Errorf("read scenario process root: %w", err)
	}
	readRecords := s.readRecords
	if readRecords == nil {
		readRecords = process.ReadScenarioRecords
	}
	owners := make([]Owner, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		records, err := readRecords(s.home, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read process records for %s: %w", entry.Name(), err)
		}
		for _, record := range records {
			owners = append(owners, Owner{
				Kind:      OwnerKindScenario,
				Name:      entry.Name(),
				PID:       record.PID,
				StartedAt: record.StartedAt,
			})
		}
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].Name == owners[j].Name {
			return owners[i].PID < owners[j].PID
		}
		return owners[i].Name < owners[j].Name
	})
	return owners, nil
}
