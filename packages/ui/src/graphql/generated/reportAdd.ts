/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportAddInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: reportAdd
// ====================================================

export interface reportAdd_reportAdd {
  __typename: "Report";
  id: string | null;
  reason: string;
  details: string | null;
}

export interface reportAdd {
  reportAdd: reportAdd_reportAdd;
}

export interface reportAddVariables {
  input: ReportAddInput;
}
