package skillscripts

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"research-desk/internal/home"
	"research-desk/internal/testutil"
)

// No-growth (ruling B5): what a scheduler and an owner are told to read at
// start and at each unit stays within the ceilings recorded in limits.json.
// The read sets and the skill-text sizes are what the desk itself reads
// (Paths.Ceilings, rulings E2). The charter is the one packet.py renders now
// from internal/testutil's niche-book-production mission; the other
// commission files are the sizes limits.json records as measured on
// programme-three, because the research writes them, not the skill. Skill
// file ceilings are the skill's own test (scripts/test_packet.py).

const (
	skillsDir  = "../../../skills"
	measuredOn = "programme-three-20261111"
)

func renderedCharter(t *testing.T) string {
	t.Helper()
	p := Packets{Paths: Paths{Skills: skillsDir}, Runner: NewPython(), ResearchDir: testutil.NewHome(t).Research}
	spec := `{"topics":["niche-book-production"],"allowance_hours":12,"stance_quotes":["Books first."],"slug":"programme-five","window":"fifth"}`
	result, err := p.Preview(context.Background(), []byte(spec))
	if err != nil {
		t.Fatalf("packet.py preview: %v", err)
	}
	if !result.OK || result.Texts == nil {
		t.Fatalf("packet.py refused the fixed spec: %+v", result.Errors)
	}
	return result.Texts.Charter
}

// measured is limits.json's read_sets.measured_before for one commission.
func measured(t *testing.T, path, commission string) map[string]int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var limits struct {
		ReadSets struct {
			MeasuredBefore map[string]json.RawMessage `json:"measured_before"`
		} `json:"read_sets"`
	}
	if err := json.Unmarshal(data, &limits); err != nil {
		t.Fatal(err)
	}
	sizes := map[string]int{}
	if err := json.Unmarshal(limits.ReadSets.MeasuredBefore[commission], &sizes); err != nil {
		t.Fatalf("%s records no measured sizes for %s: %v", path, commission, err)
	}
	return sizes
}

// [REQ:RD-P0-007]
func TestReadSetsStayWithinTheirCeilings(t *testing.T) {
	paths := Paths{Skills: skillsDir}
	ceilings, err := paths.Ceilings()
	if err != nil {
		t.Fatal(err)
	}
	for item, why := range ceilings.Unmeasured {
		t.Errorf("limits.json names %s, which cannot be measured: %s", item, why)
	}
	recorded := measured(t, ceilings.Source, measuredOn)
	charter := len(renderedCharter(t))
	size := func(item string) int {
		switch name := home.CommissionItem(item); {
		case name == home.CharterFile:
			return charter
		case name != "":
			n, ok := recorded[item]
			if !ok {
				t.Fatalf("read set names the commission file %s, but limits.json records no measured size for it on %s", item, measuredOn)
			}
			return n
		}
		return int(ceilings.Fixed[item].Bytes)
	}
	for _, role := range ceilings.Roles {
		if len(role.Items) == 0 || role.CeilingBytes <= 0 {
			t.Errorf("%s: limits.json has no read set or ceiling", role.Key)
			continue
		}
		total, parts := 0, []string{}
		for _, item := range role.Items {
			n := size(item)
			total += n
			parts = append(parts, item+" "+strconv.Itoa(n))
		}
		t.Logf("%s: %d of %d bytes (%s)", role.Key, total, role.CeilingBytes, strings.Join(parts, ", "))
		if int64(total) > role.CeilingBytes {
			t.Errorf("%s reads %d bytes, over its %d-byte ceiling (%s); cut text before raising a ceiling", role.Key, total, role.CeilingBytes, strings.Join(parts, ", "))
		}
	}
	if charter > ceilings.CharterBytes {
		t.Errorf("the rendered charter is %d bytes, over the %d-byte gate", charter, ceilings.CharterBytes)
	}
}
