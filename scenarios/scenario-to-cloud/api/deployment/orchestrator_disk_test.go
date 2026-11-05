package deployment

import (
	"testing"

	"scenario-to-cloud/domain"
)

func TestDiskDeficitWithinToleranceRequiresOnlyDiskFailure(t *testing.T) {
	resp := domain.PreflightResponse{Checks: []domain.PreflightCheck{
		{ID: domain.PreflightDiskFreeID, Status: domain.PreflightFail, Data: map[string]string{"free_kb": "5208904", "required_min_kb": "5242880"}},
	}}
	if !diskDeficitWithinTolerance(resp, 256*1024) {
		t.Fatal("expected small disk deficit to be tolerated after cleanup")
	}
	resp.Checks = append(resp.Checks, domain.PreflightCheck{ID: domain.PreflightRAMTotalID, Status: domain.PreflightFail})
	if diskDeficitWithinTolerance(resp, 256*1024) {
		t.Fatal("non-disk preflight failures must remain hard failures")
	}
}

func TestDiskDeficitBeyondToleranceRemainsHardFailure(t *testing.T) {
	resp := domain.PreflightResponse{Checks: []domain.PreflightCheck{
		{ID: domain.PreflightDiskFreeID, Status: domain.PreflightFail, Data: map[string]string{"free_kb": "4900000", "required_min_kb": "5242880"}},
	}}
	if diskDeficitWithinTolerance(resp, 256*1024) {
		t.Fatal("large disk deficit must remain a hard failure")
	}
}
