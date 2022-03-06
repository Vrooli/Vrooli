/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: projectFields
// ====================================================

export interface projectFields_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface projectFields_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projectFields_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectFields_owner = projectFields_owner_Organization | projectFields_owner_User;

export interface projectFields {
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
  tags: projectFields_tags[];
  owner: projectFields_owner | null;
}
