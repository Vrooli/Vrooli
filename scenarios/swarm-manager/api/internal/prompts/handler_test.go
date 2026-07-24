package prompts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
)

// slowSkillClient answers each skill read after a fixed delay and records the
// highest number of reads in flight at once.
type slowSkillClient struct {
	delay     time.Duration
	inFlight  atomic.Int64
	maxInFlig atomic.Int64
	failFor   string
}

func (c *slowSkillClient) GetSkill(_ context.Context, skillID string) (promptmanager.PromptSkill, error) {
	current := c.inFlight.Add(1)
	for {
		peak := c.maxInFlig.Load()
		if current <= peak || c.maxInFlig.CompareAndSwap(peak, current) {
			break
		}
	}
	time.Sleep(c.delay)
	c.inFlight.Add(-1)
	if skillID == c.failFor {
		return promptmanager.PromptSkill{}, errStatus404{}
	}
	return promptmanager.PromptSkill{ID: skillID, Name: skillID, Content: ""}, nil
}

type errStatus404 struct{}

func (errStatus404) Error() string { return "prompt-manager returned status 404" }

func (c *slowSkillClient) ReadSkill(context.Context, string, map[string]string, bool) (string, error) {
	return "", nil
}

func (c *slowSkillClient) ReadSkillWithExperiment(context.Context, string, map[string]string, bool, string) (promptmanager.ReadSkillResult, error) {
	return promptmanager.ReadSkillResult{}, nil
}

func (c *slowSkillClient) ListSkills(context.Context, string) ([]promptmanager.PromptSkill, error) {
	return nil, nil
}

func (c *slowSkillClient) UpdateSkill(context.Context, string, promptmanager.PromptSkillUpdate) (promptmanager.PromptSkill, error) {
	return promptmanager.PromptSkill{}, nil
}

func (c *slowSkillClient) GetSkillVersions(context.Context, string) (promptmanager.PromptSkillVersions, error) {
	return promptmanager.PromptSkillVersions{}, nil
}

func (c *slowSkillClient) RevertSkillVersion(context.Context, string, int) error { return nil }

func listSkills(t *testing.T, client promptmanager.AdminClient) *httptest.ResponseRecorder {
	t.Helper()
	handler := NewHandler(t.TempDir(), client)
	recorder := httptest.NewRecorder()
	handler.ListSkills(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/prompts/skills", nil))
	return recorder
}

// The catalog reads are independent, so the endpoint costs one round trip
// rather than the sum of them, and the response still follows catalog order.
func TestListSkillsReadsCatalogConcurrentlyAndPreservesOrder(t *testing.T) {
	entries := promptcatalog.SkillEntries()
	if len(entries) < 2 {
		t.Skipf("catalog has %d skills; concurrency is unobservable", len(entries))
	}
	client := &slowSkillClient{delay: 40 * time.Millisecond}

	start := time.Now()
	recorder := listSkills(t, client)
	elapsed := time.Since(start)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Items []PromptSkillSummary `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != len(entries) {
		t.Fatalf("items = %d, want %d", len(body.Items), len(entries))
	}
	for index, entry := range entries {
		if body.Items[index].ID != entry.SkillID {
			t.Fatalf("item[%d] = %q, want %q; catalog order must survive concurrent reads", index, body.Items[index].ID, entry.SkillID)
		}
	}
	if peak := client.maxInFlig.Load(); peak < 2 {
		t.Fatalf("peak concurrent reads = %d; the catalog reads are still serialized", peak)
	}
	// Serial execution would take len(entries) delays; allow generous slack so
	// this asserts "not serial" rather than a specific scheduling outcome.
	serial := time.Duration(len(entries)) * client.delay
	if elapsed >= serial {
		t.Fatalf("elapsed %s is at least the serial cost %s", elapsed, serial)
	}
}

// One failing read fails the whole projection rather than serving a summary
// with a silently empty entry.
func TestListSkillsSurfacesAFailedRead(t *testing.T) {
	entries := promptcatalog.SkillEntries()
	if len(entries) == 0 {
		t.Skip("catalog has no skills")
	}
	client := &slowSkillClient{failFor: entries[len(entries)-1].SkillID}

	recorder := listSkills(t, client)

	if recorder.Code == http.StatusOK {
		t.Fatalf("a failed skill read produced a 200 response: %s", recorder.Body.String())
	}
}
