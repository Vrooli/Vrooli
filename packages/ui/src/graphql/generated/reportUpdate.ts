/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: reportUpdate
// ====================================================

export interface reportUpdate_reportUpdate {
  __typename: "Report";
  id: string | null;
  reason: string;
  details: string | null;
}

export interface reportUpdate {
  reportUpdate: reportUpdate_reportUpdate;
}

export interface reportUpdateVariables {
  input: ReportUpdateInput;
}
