/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectAdd
// ====================================================

export interface projectAdd_projectAdd_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
}

export interface projectAdd_projectAdd {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectAdd_projectAdd_tags[];
  stars: number;
}

export interface projectAdd {
  projectAdd: projectAdd_projectAdd;
}

export interface projectAddVariables {
  input: ProjectInput;
}
