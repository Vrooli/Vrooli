/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardReport
// ====================================================

export interface standardReport_standardReport {
  __typename: "Success";
  success: boolean | null;
}

export interface standardReport {
  standardReport: standardReport_standardReport;
}

export interface standardReportVariables {
  input: ReportInput;
}
