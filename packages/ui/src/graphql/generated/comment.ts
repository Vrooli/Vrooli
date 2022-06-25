/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: comment
// ====================================================

export interface comment_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comment_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: comment_comment_commentedOn_Project_translations[];
}

export interface comment_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface comment_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  translations: comment_comment_commentedOn_Routine_translations[];
}

export interface comment_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
  type: string;
}

export type comment_comment_commentedOn = comment_comment_commentedOn_Project | comment_comment_commentedOn_Routine | comment_comment_commentedOn_Standard;

export interface comment_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comment_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: comment_comment_creator_Organization_translations[];
}

export interface comment_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type comment_comment_creator = comment_comment_creator_Organization | comment_comment_creator_User;

export interface comment_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface comment_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: comment_comment_commentedOn;
  creator: comment_comment_creator | null;
  translations: comment_comment_translations[];
}

export interface comment {
  comment: comment_comment | null;
}

export interface commentVariables {
  input: FindByIdInput;
}
