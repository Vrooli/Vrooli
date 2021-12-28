/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationReport
// ====================================================

export interface organizationReport_organizationReport {
  __typename: "Success";
  success: boolean | null;
}

export interface organizationReport {
  organizationReport: organizationReport_organizationReport;
}

export interface organizationReportVariables {
  input: ReportInput;
}
