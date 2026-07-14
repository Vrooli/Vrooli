package baseline

import (
	"reflect"
	"testing"
)

func TestParseCollectionMemberUsesCollectionNameAndAllowsOverride(t *testing.T) {
	member, err := parseCollectionMember("plan-manager", "before")
	if err != nil || member.GetScenario() != "plan-manager" || member.GetBaselineName() != "before" || !member.GetRequired() {
		t.Fatalf("default member = %#v err=%v", member, err)
	}
	member, err = parseCollectionMember("git-control-tower:separate", "before")
	if err != nil || member.GetBaselineName() != "separate" {
		t.Fatalf("override member = %#v err=%v", member, err)
	}
	if _, err := parseCollectionMember(":bad", "before"); err == nil {
		t.Fatal("empty scenario accepted")
	}
}

func TestCollectionFollowupArgsOmitEmptyBranch(t *testing.T) {
	got := collectionFollowupArgs("before", "", "--wait")
	want := []string{"--name", "before", "--wait"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("follow-up args = %#v, want %#v", got, want)
	}
}
