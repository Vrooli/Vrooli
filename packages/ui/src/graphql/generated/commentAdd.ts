/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentAdd
// ====================================================

export interface commentAdd_commentAdd_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface commentAdd_commentAdd_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type commentAdd_commentAdd_creator = commentAdd_commentAdd_creator_Organization | commentAdd_commentAdd_creator_User;

export interface commentAdd_commentAdd_commentedOn_Project {
  __typename: "Project";
  id: string;
  name: string;
}

export interface commentAdd_commentAdd_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface commentAdd_commentAdd_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type commentAdd_commentAdd_commentedOn = commentAdd_commentAdd_commentedOn_Project | commentAdd_commentAdd_commentedOn_Routine | commentAdd_commentAdd_commentedOn_Standard;

export interface commentAdd_commentAdd {
  __typename: "Comment";
  id: string;
  text: string | null;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  isStarred: boolean | null;
  creator: commentAdd_commentAdd_creator | null;
  commentedOn: commentAdd_commentAdd_commentedOn;
}

export interface commentAdd {
  commentAdd: commentAdd_commentAdd;
}

export interface commentAddVariables {
  input: CommentInput;
}
