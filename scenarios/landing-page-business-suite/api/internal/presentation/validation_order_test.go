package presentation

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestValidateOrdersAmbiguousPageOwnerIssues(t *testing.T) {
	var apps []App
	want := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		pageID := fmt.Sprintf("page-%02d", i/2)
		apps = append(apps, testApp(
			fmt.Sprintf("app-%02d", i), fmt.Sprintf("app-%02d", i), pageID,
			VisibilityPublic, PublicationDraft, true,
		))
	}
	for i := 0; i < 4; i++ {
		want = append(want, fmt.Sprintf("pages[page-%02d]", i))
	}
	document := testDocument(apps...)

	for attempt := 0; attempt < 100; attempt++ {
		err := Validate(document)
		var validation *ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("Validate error = %v, want ValidationError", err)
		}
		var got []string
		for _, issue := range validation.Issues {
			if issue.Code == "ambiguous_page_owner" {
				got = append(got, issue.Path)
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ambiguous owner issue order = %v, want %v (attempt %d)", got, want, attempt)
		}
	}
}

func TestValidateThemeOrdersTokenIssues(t *testing.T) {
	issues := &ValidationError{}
	validateTheme(Theme{}, "theme", issues)

	var got []string
	for _, issue := range issues.Issues {
		got = append(got, issue.Path)
	}
	want := []string{"theme.variant", "theme.accent", "theme.background", "theme.primary"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("theme issue order = %v, want %v", got, want)
	}
}
