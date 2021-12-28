/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineReport
// ====================================================

export interface routineReport_routineReport {
  __typename: "Success";
  success: boolean | null;
}

export interface routineReport {
  routineReport: routineReport_routineReport;
}

export interface routineReportVariables {
  input: ReportInput;
}
