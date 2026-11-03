package main

import (
	"landing-page-business-suite-api/internal/commerce"
	intelligencecore "landing-page-business-suite-api/internal/intelligence"
)

func commerceReport(report intelligencecore.UsageReport) commerce.UsageReportRequest {
	return commerce.UsageReportRequest{UserIdentity: report.UserIdentity, LimitKey: report.LimitKey, Amount: report.Amount, AppBundleKey: report.AppBundleKey, Operation: report.Operation, Metadata: report.Metadata, OperationID: report.OperationID}
}
