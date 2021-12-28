/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceReport
// ====================================================

export interface resourceReport_resourceReport {
  __typename: "Success";
  success: boolean | null;
}

export interface resourceReport {
  resourceReport: resourceReport_resourceReport;
}

export interface resourceReportVariables {
  input: ReportInput;
}
