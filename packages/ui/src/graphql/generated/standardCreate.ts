/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardCreateInput, MemberRole, StandardType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardCreate
// ====================================================

export interface standardCreate_standardCreate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface standardCreate_standardCreate_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface standardCreate_standardCreate_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type standardCreate_standardCreate_creator = standardCreate_standardCreate_creator_Organization | standardCreate_standardCreate_creator_User;

export interface standardCreate_standardCreate {
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
  tags: standardCreate_standardCreate_tags[];
  creator: standardCreate_standardCreate_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardCreate {
  standardCreate: standardCreate_standardCreate;
}

export interface standardCreateVariables {
  input: StandardCreateInput;
}
