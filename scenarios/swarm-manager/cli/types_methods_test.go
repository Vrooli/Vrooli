package main

import "testing"

func TestUpdateBacklogRequestEmpty(t *testing.T) {
	if !(UpdateBacklogRequest{}).Empty() {
		t.Error("zero-value UpdateBacklogRequest should be Empty")
	}

	title := "x"
	if (UpdateBacklogRequest{Title: &title}).Empty() {
		t.Error("request with Title should not be Empty")
	}

	prio := 3
	if (UpdateBacklogRequest{Priority: &prio}).Empty() {
		t.Error("request with Priority should not be Empty")
	}

	tags := []string{}
	if (UpdateBacklogRequest{Tags: &tags}).Empty() {
		t.Error("request with non-nil empty Tags pointer should not be Empty (array-clear)")
	}

	creates := []string{"a"}
	if (UpdateBacklogRequest{Creates: &creates}).Empty() {
		t.Error("request with Creates should not be Empty")
	}
}

func TestInitiativeUpdateRequestHasChanges(t *testing.T) {
	if (InitiativeUpdateRequest{}).HasChanges() {
		t.Error("zero-value should have no changes")
	}

	status := "active"
	if !(InitiativeUpdateRequest{Status: &status}).HasChanges() {
		t.Error("Status set should report changes")
	}

	items := []string{}
	if !(InitiativeUpdateRequest{Items: &items}).HasChanges() {
		t.Error("non-nil Items pointer should report changes")
	}

	prio := 0
	if !(InitiativeUpdateRequest{Priority: &prio}).HasChanges() {
		t.Error("Priority pointer to zero should still report changes")
	}
}
