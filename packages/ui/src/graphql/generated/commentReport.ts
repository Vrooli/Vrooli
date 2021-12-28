/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentReport
// ====================================================

export interface commentReport_commentReport {
  __typename: "Success";
  success: boolean | null;
}

export interface commentReport {
  commentReport: commentReport_commentReport;
}

export interface commentReportVariables {
  input: ReportInput;
}
