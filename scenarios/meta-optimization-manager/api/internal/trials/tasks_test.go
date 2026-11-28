package trials

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
)

type fixedSpaceReader struct {
	definition *spacedoc.SpaceDefinition
	err        error
	projection spacedoc.Projection
}

func (r *fixedSpaceReader) Read(_ context.Context, projection spacedoc.Projection) (*spacedoc.SpaceDefinition, error) {
	r.projection = projection
	return r.definition, r.err
}

func TestTaskGeneratorBuildsPositiveAndDeterministicHonestyTasks(t *testing.T) {
	reader := &fixedSpaceReader{definition: &spacedoc.SpaceDefinition{Cells: []spacedoc.Cell{
		{ID: "g-research", Group: "RESEARCH", Question: "Compare two reports"},
		{ID: "g-read", Group: "Reading", Question: "Explain the state"},
		{ID: "g-bug", Group: "Operations", Question: "Repair a broken run"},
		{ID: "g-feature", Group: "General", Question: "Add a dashboard"},
		{ID: "g-research-2", Group: "General", Question: "Investigate another source"},
	}}}
	generator := NewTaskGenerator(reader)

	got, err := generator.Generate(context.Background(), "")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if reader.projection != spacedoc.ProjectionGuide {
		t.Fatalf("projection = %q, want Guide", reader.projection)
	}
	wantPositiveSuites := []string{"research", "comprehend", "bugfix", "add-feature", "research"}
	wantQuestions := []string{"Compare two reports", "Explain the state", "Repair a broken run", "Add a dashboard", "Investigate another source"}
	if len(got) != len(wantPositiveSuites)+4 {
		t.Fatalf("generated %d tasks, want %d positives and one honesty case per family", len(got), len(wantPositiveSuites)+4)
	}
	for i, wantSuite := range wantPositiveSuites {
		wantID := []string{"g-research", "g-read", "g-bug", "g-feature", "g-research-2"}[i]
		if got[i].ID != "trial/"+wantID || got[i].GuideTaskID != wantID || got[i].Suite != wantSuite || got[i].Negative || got[i].Description != wantQuestions[i] {
			t.Errorf("positive task %d = %+v, want id=%q suite=%q with source question", i, got[i], wantID, wantSuite)
		}
	}
	wantNegativeIDs := []string{
		"trial/negative/add-feature",
		"trial/negative/bugfix",
		"trial/negative/comprehend",
		"trial/negative/research",
	}
	for i, wantID := range wantNegativeIDs {
		gotTask := got[len(wantPositiveSuites)+i]
		if gotTask.ID != wantID || gotTask.GuideTaskID != wantID[len("trial/"):] || gotTask.Suite != "negative" || !gotTask.Negative {
			t.Errorf("honesty task %d = %+v, want id=%q, negative suite and flag", i, gotTask, wantID)
		}
	}

	filtered, err := generator.Generate(context.Background(), "negative")
	if err != nil {
		t.Fatalf("Generate(negative) error = %v", err)
	}
	var filteredIDs []string
	for _, task := range filtered {
		filteredIDs = append(filteredIDs, task.ID)
	}
	if !reflect.DeepEqual(filteredIDs, wantNegativeIDs) {
		t.Fatalf("negative suite IDs = %v, want %v", filteredIDs, wantNegativeIDs)
	}
}

func TestTaskGeneratorHandlesEmptyGuideAndReaderErrors(t *testing.T) {
	empty, err := NewTaskGenerator(&fixedSpaceReader{definition: &spacedoc.SpaceDefinition{}}).Generate(context.Background(), "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty Guide tasks = %v, err=%v; want no tasks and no error", empty, err)
	}

	wantErr := errors.New("Guide read unavailable")
	_, err = NewTaskGenerator(&fixedSpaceReader{err: wantErr}).Generate(context.Background(), "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("reader error = %v, want wrapped %v", err, wantErr)
	}
}
