/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole, StandardType } from "./globalTypes";

// ====================================================
// GraphQL query operation: standard
// ====================================================

export interface standard_standard_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface standard_standard_creator_Organization_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface standard_standard_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
  bio: string | null;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  tags: standard_standard_creator_Organization_tags[];
}

export interface standard_standard_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
  bio: string | null;
}

export type standard_standard_creator = standard_standard_creator_Organization | standard_standard_creator_User;

export interface standard_standard {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  role: MemberRole | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standard_standard_tags[];
  creator: standard_standard_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface standard {
  standard: standard_standard | null;
}

export interface standardVariables {
  input: FindByIdInput;
}
