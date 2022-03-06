/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectUpdateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectUpdate
// ====================================================

export interface projectUpdate_projectUpdate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
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
  tags: projectUpdate_projectUpdate_tags[];
  owner: projectUpdate_projectUpdate_owner | null;
}

export interface projectUpdate {
  projectUpdate: projectUpdate_projectUpdate;
}

export interface projectUpdateVariables {
  input: ProjectUpdateInput;
}
