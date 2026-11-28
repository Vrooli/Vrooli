package capacity

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

// SupportProfile is the RAM and CPU a support job declares to the host ledger
// while it runs. Work that is mostly waiting on the network or disk declares
// what it really uses, so a busy host does not hold it behind a full-core claim
// it never needs.
type SupportProfile struct {
	RAMBytes int64  `json:"ram_bytes"`
	CPUMilli int64  `json:"cpu_milli"`
	Reason   string `json:"reason"`
}

type supportProfileTable struct {
	Default    SupportProfile            `json:"default"`
	Operations map[string]SupportProfile `json:"operations"`
}

//go:embed support_profiles.json
var supportProfilesJSON []byte

var supportProfiles = mustLoadSupportProfiles(supportProfilesJSON)

func mustLoadSupportProfiles(data []byte) supportProfileTable {
	var table supportProfileTable
	if err := json.Unmarshal(data, &table); err != nil {
		panic(fmt.Sprintf("capacity: support_profiles.json: %v", err))
	}
	if err := table.validate(); err != nil {
		panic(err.Error())
	}
	return table
}

func (t supportProfileTable) validate() error {
	check := func(name string, p SupportProfile) error {
		if p.RAMBytes <= 0 || p.CPUMilli <= 0 || p.Reason == "" {
			return fmt.Errorf("capacity: support profile %q needs positive ram_bytes, cpu_milli and a reason", name)
		}
		return nil
	}
	if err := check("default", t.Default); err != nil {
		return err
	}
	for name, p := range t.Operations {
		if err := check(name, p); err != nil {
			return err
		}
	}
	return nil
}

// SupportProfileFor returns the declared profile for an operation, or the
// default when it has none.
func SupportProfileFor(operation string) SupportProfile {
	if p, ok := supportProfiles.Operations[operation]; ok {
		return p
	}
	return supportProfiles.Default
}

// SupportProfileOperations lists the operations with their own profile.
func SupportProfileOperations() []string {
	out := make([]string, 0, len(supportProfiles.Operations))
	for name := range supportProfiles.Operations {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
