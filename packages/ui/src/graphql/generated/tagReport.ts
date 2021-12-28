/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagReport
// ====================================================

export interface tagReport_tagReport {
  __typename: "Success";
  success: boolean | null;
}

export interface tagReport {
  tagReport: tagReport_tagReport;
}

export interface tagReportVariables {
  input: ReportInput;
}
