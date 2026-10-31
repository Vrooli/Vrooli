package uxmetrics

import "errors"

var (
	errInvalidExecutionID = errors.New("invalid execution id")
	errInvalidWorkflowID  = errors.New("invalid workflow id")
	errInvalidStepIndex   = errors.New("invalid step index")
	errMetricsNotFound    = errors.New("metrics not found for this execution")
	errStepMetricsMissing = errors.New("step metrics not found")
	errProTierRequired    = errors.New("UX metrics requires Pro plan or higher")
)
