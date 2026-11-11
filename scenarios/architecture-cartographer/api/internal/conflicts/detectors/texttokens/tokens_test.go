package texttokens

import (
	"reflect"
	"testing"
)

func TestExtractSplitsCamelCaseAndPathsWithDetectorSpecificStops(t *testing.T) {
	ignored := map[string]struct{}{"api": {}, "handlers": {}}
	got := Extract("API/UserProfile/api_handlers/GraphNode", ignored)
	want := []string{"graph", "node", "profile", "user"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Extract() = %v, want %v", got, want)
	}
}

func TestExtractKeepsUnignoredVocabulary(t *testing.T) {
	got := Extract("RequestHandler", nil)
	want := []string{"handler", "request"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Extract() = %v, want %v", got, want)
	}
}
