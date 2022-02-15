/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentCreate
// ====================================================

export interface commentCreate_commentCreate_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface commentCreate_commentCreate_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type commentCreate_commentCreate_creator = commentCreate_commentCreate_creator_Organization | commentCreate_commentCreate_creator_User;

export interface commentCreate_commentCreate_commentedOn_Project {
  __typename: "Project";
  id: string;
  name: string;
}

export interface commentCreate_commentCreate_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface commentCreate_commentCreate_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type commentCreate_commentCreate_commentedOn = commentCreate_commentCreate_commentedOn_Project | commentCreate_commentCreate_commentedOn_Routine | commentCreate_commentCreate_commentedOn_Standard;

export interface commentCreate_commentCreate {
  __typename: "Comment";
  id: string;
  text: string | null;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  creator: commentCreate_commentCreate_creator | null;
  commentedOn: commentCreate_commentCreate_commentedOn;
}

export interface commentCreate {
  commentCreate: commentCreate_commentCreate;
}

export interface commentCreateVariables {
  input: CommentCreateInput;
}
