/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: project
// ====================================================

export interface project_project {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
}

export interface project {
  project: project_project | null;
}

export interface projectVariables {
  input: FindByIdInput;
}
