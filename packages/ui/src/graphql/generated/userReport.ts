/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: userReport
// ====================================================

export interface userReport_userReport {
  __typename: "Success";
  success: boolean | null;
}

export interface userReport {
  userReport: userReport_userReport;
}

export interface userReportVariables {
  input: ReportInput;
}
