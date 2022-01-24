/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: project
// ====================================================

export interface project_project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface project_project_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface project_project_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type project_project_owner = project_project_owner_Organization | project_project_owner_User;

export interface project_project {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: project_project_tags[];
  owner: project_project_owner | null;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface project {
  project: project_project | null;
}

export interface projectVariables {
  input: FindByIdInput;
}
