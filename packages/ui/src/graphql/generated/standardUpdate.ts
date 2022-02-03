/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardUpdateInput, StandardType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardUpdate
// ====================================================

export interface standardUpdate_standardUpdate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface standardUpdate_standardUpdate_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface standardUpdate_standardUpdate_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type standardUpdate_standardUpdate_creator = standardUpdate_standardUpdate_creator_Organization | standardUpdate_standardUpdate_creator_User;

export interface standardUpdate_standardUpdate {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standardUpdate_standardUpdate_tags[];
  creator: standardUpdate_standardUpdate_creator | null;
  stars: number;
  isStarred: boolean | null;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardUpdate {
  standardUpdate: standardUpdate_standardUpdate;
}

export interface standardUpdateVariables {
  input: StandardUpdateInput;
}
