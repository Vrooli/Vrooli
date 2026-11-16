package transformation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"music-tools/internal/jobs"
)

func validJobAndReceipt() (jobs.Job, DerivationReceipt) {
	job := jobs.Job{
		ID: "job-1", Operation: "transform.sample", State: jobs.StateSucceeded,
		ResultRef: "outputs/job-1/derived.wav",
	}
	receipt := DerivationReceipt{
		SchemaVersion: "1", JobID: job.ID, Kind: "sample",
		SourceTrackSHA256:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DerivedTrackSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SourceStartSeconds: 2, SourceEndSeconds: 6, DerivedStartSeconds: 10, DerivedEndSeconds: 14,
		ResultRef: job.ResultRef, OperationVersion: "1", ToolName: "ffmpeg",
		ToolVersion: "7.0", InheritedRightsScope: "personal_only",
	}
	return job, receipt
}

func TestReadJobReceiptAcceptsMatchingSuccessfulSampleJob(t *testing.T) {
	job, receipt := validJobAndReceipt()
	encoded, err := json.Marshal(receipt)
	require.NoError(t, err)
	job.Meta = map[string]string{ReceiptMetaKey: string(encoded)}

	actual, err := ReadJobReceipt(job)
	require.NoError(t, err)
	require.Equal(t, receipt, actual)
}

func TestReadJobReceiptRejectsUntrustedOrMismatchedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*jobs.Job, *DerivationReceipt)
	}{
		{name: "nonterminal", mutate: func(job *jobs.Job, _ *DerivationReceipt) { job.State = jobs.StateRunning }},
		{name: "nontransformation operation", mutate: func(job *jobs.Job, _ *DerivationReceipt) { job.Operation = "compose" }},
		{name: "wrong job", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.JobID = "job-other" }},
		{name: "wrong kind", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.Kind = "remix" }},
		{name: "bad output hash", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.DerivedTrackSHA256 = "not-a-hash" }},
		{name: "same source and output", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.DerivedTrackSHA256 = receipt.SourceTrackSHA256 }},
		{name: "empty excerpt", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.SourceEndSeconds = receipt.SourceStartSeconds }},
		{name: "unbound result", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.ResultRef = "outputs/other.wav" }},
		{name: "rights widening claim", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.InheritedRightsScope = "unrestricted" }},
		{name: "model without revision", mutate: func(_ *jobs.Job, receipt *DerivationReceipt) { receipt.ModelID = "model-a" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job, receipt := validJobAndReceipt()
			tc.mutate(&job, &receipt)
			encoded, err := json.Marshal(receipt)
			require.NoError(t, err)
			job.Meta = map[string]string{ReceiptMetaKey: string(encoded)}
			_, err = ReadJobReceipt(job)
			require.Error(t, err)
		})
	}
}

func TestReadJobReceiptRejectsMissingMalformedAndTrailingData(t *testing.T) {
	job, _ := validJobAndReceipt()
	for _, raw := range []string{"", "{}", `{"schema_version":"1"} {}`} {
		job.Meta = map[string]string{ReceiptMetaKey: raw}
		_, err := ReadJobReceipt(job)
		require.Error(t, err)
	}
}

func TestEncodeJobReceiptBindsJobAndOutputReference(t *testing.T) {
	job, receipt := validJobAndReceipt()
	encoded, err := EncodeJobReceipt(job.ID, job.Operation, job.ResultRef, receipt)
	require.NoError(t, err)
	job.Meta = map[string]string{ReceiptMetaKey: encoded}
	_, err = ReadJobReceipt(job)
	require.NoError(t, err)

	_, err = EncodeJobReceipt(job.ID, job.Operation, "outputs/other.wav", receipt)
	require.Error(t, err)
}
