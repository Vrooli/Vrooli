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
  isStarred: boolean | null;
}

export interface projectAdd_projectAdd_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projectAdd_projectAdd_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectAdd_projectAdd_owner = projectAdd_projectAdd_owner_Organization | projectAdd_projectAdd_owner_User;

export interface projectAdd_projectAdd {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectAdd_projectAdd_tags[];
  owner: projectAdd_projectAdd_owner | null;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface projectAdd {
  projectAdd: projectAdd_projectAdd;
}

export interface projectAddVariables {
  input: ProjectInput;
}
