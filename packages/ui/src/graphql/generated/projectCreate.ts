/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectCreate
// ====================================================

export interface projectCreate_projectCreate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
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
  completedAt: any | null;
  created_at: any;
  description: string | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  name: string;
  role: MemberRole | null;
  score: number;
  stars: number;
  tags: projectCreate_projectCreate_tags[];
  owner: projectCreate_projectCreate_owner | null;
}

export interface projectCreate {
  projectCreate: projectCreate_projectCreate;
}

export interface projectCreateVariables {
  input: ProjectCreateInput;
}
