/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectReport
// ====================================================

export interface projectReport_projectReport {
  __typename: "Success";
  success: boolean | null;
}

export interface projectReport {
  projectReport: projectReport_projectReport;
}

export interface projectReportVariables {
  input: ReportInput;
}
