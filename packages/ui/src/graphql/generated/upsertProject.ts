/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: upsertProject
// ====================================================

export interface upsertProject_upsertProject {
  __typename: "Project";
  id: string;
}

export interface upsertProject {
  upsertProject: upsertProject_upsertProject;
}

export interface upsertProjectVariables {
  input: ProjectInput;
}
