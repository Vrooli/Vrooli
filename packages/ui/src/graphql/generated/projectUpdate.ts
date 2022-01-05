/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectUpdate
// ====================================================

export interface projectUpdate_projectUpdate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface projectUpdate_projectUpdate {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectUpdate_projectUpdate_tags[];
}

export interface projectUpdate {
  projectUpdate: projectUpdate_projectUpdate;
}

export interface projectUpdateVariables {
  input: ProjectInput;
}
