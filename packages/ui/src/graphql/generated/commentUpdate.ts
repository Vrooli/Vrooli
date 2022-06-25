/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentUpdateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentUpdate
// ====================================================

export interface commentUpdate_commentUpdate_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentUpdate_commentUpdate_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: commentUpdate_commentUpdate_commentedOn_Project_translations[];
}

export interface commentUpdate_commentUpdate_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface commentUpdate_commentUpdate_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  translations: commentUpdate_commentUpdate_commentedOn_Routine_translations[];
}

export interface commentUpdate_commentUpdate_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
  type: string;
}

export type commentUpdate_commentUpdate_commentedOn = commentUpdate_commentUpdate_commentedOn_Project | commentUpdate_commentUpdate_commentedOn_Routine | commentUpdate_commentUpdate_commentedOn_Standard;

export interface commentUpdate_commentUpdate_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentUpdate_commentUpdate_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: commentUpdate_commentUpdate_creator_Organization_translations[];
}

export interface commentUpdate_commentUpdate_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type commentUpdate_commentUpdate_creator = commentUpdate_commentUpdate_creator_Organization | commentUpdate_commentUpdate_creator_User;

export interface commentUpdate_commentUpdate_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface commentUpdate_commentUpdate {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: commentUpdate_commentUpdate_commentedOn;
  creator: commentUpdate_commentUpdate_creator | null;
  translations: commentUpdate_commentUpdate_translations[];
}

export interface commentUpdate {
  commentUpdate: commentUpdate_commentUpdate;
}

export interface commentUpdateVariables {
  input: CommentUpdateInput;
}
