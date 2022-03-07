/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ReportCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: reportCreate
// ====================================================

export interface reportCreate_reportCreate {
  __typename: "Report";
  id: string | null;
  language: string;
  reason: string;
  details: string | null;
  isOwn: boolean;
}

export interface reportCreate {
  reportCreate: reportCreate_reportCreate;
}

export interface reportCreateVariables {
  input: ReportCreateInput;
}
