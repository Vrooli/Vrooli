package selection

import "testing"

func TestMenuOperationsExistInAICatalog(t *testing.T) {
	for op := range menuOps() {
		if !knownAIOp(op) {
			t.Errorf("contextual edit references unknown AI op %q", op)
		}
	}
}

func TestListClassesStableOrder(t *testing.T) {
	classes := ListClasses()
	if len(classes) != len(classOrder) {
		t.Fatalf("ListClasses returned %d, want %d", len(classes), len(classOrder))
	}
	for i, want := range classOrder {
		if classes[i].Name != want {
			t.Errorf("class[%d] = %q, want %q", i, classes[i].Name, want)
		}
		if len(classes[i].Edits) == 0 {
			t.Errorf("class %q has no edits", classes[i].Name)
		}
	}
}

func TestSuggestUnknownFallsBackToObject(t *testing.T) {
	resolved, edits := Suggest("definitely-not-a-class")
	if resolved != ClassObject {
		t.Errorf("resolved = %q, want %q", resolved, ClassObject)
	}
	if len(edits) == 0 {
		t.Error("object menu is empty")
	}
}

func TestSuggestKnownClass(t *testing.T) {
	resolved, edits := Suggest(ClassSky)
	if resolved != ClassSky {
		t.Errorf("resolved = %q, want %q", resolved, ClassSky)
	}
	if len(edits) == 0 {
		t.Error("sky menu is empty")
	}
}

func TestSuggestReturnsCopy(t *testing.T) {
	_, a := Suggest(ClassObject)
	if len(a) == 0 {
		t.Fatal("no edits")
	}
	a[0].Label = "mutated"
	_, b := Suggest(ClassObject)
	if b[0].Label == "mutated" {
		t.Error("Suggest returned a shared slice; callers can mutate the menu")
	}
}

// Every prompt-templated edit (requires_prompt) must carry a trailing-space hint
// or empty prompt the caller completes — never a fully-formed sentence that
// would read wrong once the user's text is appended.
func TestRequiresPromptEditsHaveCompletableHints(t *testing.T) {
	for _, rc := range ListClasses() {
		for _, e := range rc.Edits {
			if !e.RequiresPrompt {
				continue
			}
			if e.Prompt != "" && e.Prompt[len(e.Prompt)-1] != ' ' {
				t.Errorf("%s/%s prompt %q should end with a space (user text is appended)", rc.Name, e.ID, e.Prompt)
			}
		}
	}
}

// Whole-image ops (background_removal) must NOT require the selection mask; all
// region ops must.
func TestBackgroundRemovalDoesNotRequireMask(t *testing.T) {
	_, edits := Suggest(ClassBackground)
	found := false
	for _, e := range edits {
		if e.Operation == "background_removal" {
			found = true
			if e.RequiresMask {
				t.Error("background_removal must not require the selection mask")
			}
		}
	}
	if !found {
		t.Error("background menu should offer background_removal")
	}
}
