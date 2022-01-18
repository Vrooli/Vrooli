/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentUpdate
// ====================================================

export interface commentUpdate_commentUpdate_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface commentUpdate_commentUpdate_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type commentUpdate_commentUpdate_creator = commentUpdate_commentUpdate_creator_Organization | commentUpdate_commentUpdate_creator_User;

export interface commentUpdate_commentUpdate_commentedOn_Project {
  __typename: "Project";
  id: string;
  name: string;
}

export interface commentUpdate_commentUpdate_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface commentUpdate_commentUpdate_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type commentUpdate_commentUpdate_commentedOn = commentUpdate_commentUpdate_commentedOn_Project | commentUpdate_commentUpdate_commentedOn_Routine | commentUpdate_commentUpdate_commentedOn_Standard;

export interface commentUpdate_commentUpdate {
  __typename: "Comment";
  id: string;
  text: string | null;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean;
  creator: commentUpdate_commentUpdate_creator | null;
  commentedOn: commentUpdate_commentUpdate_commentedOn;
}

export interface commentUpdate {
  commentUpdate: commentUpdate_commentUpdate;
}

export interface commentUpdateVariables {
  input: CommentInput;
}
