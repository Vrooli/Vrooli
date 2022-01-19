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
  stars: number;
  score: number;
  isUpvoted: boolean | null;
}

export interface projectUpdate_projectUpdate_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projectUpdate_projectUpdate_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectUpdate_projectUpdate_owner = projectUpdate_projectUpdate_owner_Organization | projectUpdate_projectUpdate_owner_User;

export interface projectUpdate_projectUpdate {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectUpdate_projectUpdate_tags[];
  owner: projectUpdate_projectUpdate_owner | null;
  stars: number;
  score: number;
  isUpvoted: boolean | null;
}

export interface projectUpdate {
  projectUpdate: projectUpdate_projectUpdate;
}

export interface projectUpdateVariables {
  input: ProjectInput;
}
