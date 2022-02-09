/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectCreate
// ====================================================

export interface projectCreate_projectCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface projectCreate_projectCreate_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projectCreate_projectCreate_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectCreate_projectCreate_owner = projectCreate_projectCreate_owner_Organization | projectCreate_projectCreate_owner_User;

export interface projectCreate_projectCreate {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectCreate_projectCreate_tags[];
  owner: projectCreate_projectCreate_owner | null;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface projectCreate {
  projectCreate: projectCreate_projectCreate;
}

export interface projectCreateVariables {
  input: ProjectCreateInput;
}
