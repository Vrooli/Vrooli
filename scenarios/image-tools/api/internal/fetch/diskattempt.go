package fetch

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// attemptSegment separates a durable job key from the attempt that holds the
// lease. The ledger keeps one immutable receipt per (owner, job key), so a
// durable job that is requeued after a drain, a crash or a capacity-busy
// refusal can never reuse its key: each attempt reserves under its own.
const attemptSegment = "/attempt/"

// DiskJobSuperseder ends the previous attempts of one durable job. An
// admitter that keeps receipts implements it; the new attempt takes over the
// work, so the predecessor's reservation must not be counted a second time.
// ownsKey reports which of the owner's job keys belong to the durable job.
type DiskJobSuperseder interface {
	SupersedeDiskJobs(ctx context.Context, ownerID string, ownsKey func(jobKey string) bool) (int, error)
}

// AttemptPrefix is the key prefix shared by every attempt of one durable job.
func AttemptPrefix(jobID string) string {
	return strings.TrimSpace(jobID) + attemptSegment
}

// IsAttemptOf reports whether a ledger job key was taken by an attempt of the
// durable job: an attempt-scoped key, or the bare durable key that installs
// reserved under before attempts were scoped (a row an older process left).
func IsAttemptOf(durableJobID, jobKey string) bool {
	durableJobID = strings.TrimSpace(durableJobID)
	return durableJobID != "" && (jobKey == durableJobID || strings.HasPrefix(jobKey, AttemptPrefix(durableJobID)))
}

// ReserveAttempt reserves the request as a new attempt of its durable job.
// request.JobID names the durable job; the lease is taken under an
// attempt-scoped key after any earlier attempt whose producer is gone has been
// superseded (reattached, then completed). A resumed install therefore
// succeeds with its old lease row present, whether that row is still active,
// lapsed or completed.
func ReserveAttempt(ctx context.Context, admit sharedcapacity.DiskJobAdmitter, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	if admit == nil {
		return nil, errors.New("fetch: shared host disk admission is unavailable")
	}
	var supersedeErr error
	if s, ok := admit.(DiskJobSuperseder); ok {
		durable := request.JobID
		_, supersedeErr = s.SupersedeDiskJobs(ctx, request.OwnerID, func(key string) bool { return IsAttemptOf(durable, key) })
	}
	request.JobID = AttemptPrefix(request.JobID) + uuid.NewString()
	lease, err := admit.ReserveDiskJob(ctx, request)
	if err != nil {
		return nil, errors.Join(err, supersedeErr)
	}
	return lease, nil
}
