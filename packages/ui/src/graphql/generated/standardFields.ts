/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardType } from "./globalTypes";

// ====================================================
// GraphQL fragment: standardFields
// ====================================================

export interface standardFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  votes: number;
  isUpvoted: boolean;
}

export interface standardFields_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface standardFields_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type standardFields_creator = standardFields_creator_Organization | standardFields_creator_User;

export interface standardFields {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standardFields_tags[];
  creator: standardFields_creator | null;
  stars: number;
  votes: number;
  isUpvoted: boolean;
}
