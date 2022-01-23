/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardInput, StandardType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardAdd
// ====================================================

export interface standardAdd_standardAdd_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardAdd_standardAdd_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface standardAdd_standardAdd_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type standardAdd_standardAdd_creator = standardAdd_standardAdd_creator_Organization | standardAdd_standardAdd_creator_User;

export interface standardAdd_standardAdd {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standardAdd_standardAdd_tags[];
  creator: standardAdd_standardAdd_creator | null;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardAdd {
  standardAdd: standardAdd_standardAdd;
}

export interface standardAddVariables {
  input: StandardInput;
}
