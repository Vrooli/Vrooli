/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentCreate
// ====================================================

export interface commentCreate_commentCreate_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentCreate_commentCreate_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: commentCreate_commentCreate_commentedOn_Project_translations[];
}

export interface commentCreate_commentCreate_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface commentCreate_commentCreate_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: commentCreate_commentCreate_commentedOn_Routine_translations[];
}

export interface commentCreate_commentCreate_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type commentCreate_commentCreate_commentedOn = commentCreate_commentCreate_commentedOn_Project | commentCreate_commentCreate_commentedOn_Routine | commentCreate_commentCreate_commentedOn_Standard;

export interface commentCreate_commentCreate_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentCreate_commentCreate_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: commentCreate_commentCreate_creator_Organization_translations[];
}

export interface commentCreate_commentCreate_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type commentCreate_commentCreate_creator = commentCreate_commentCreate_creator_Organization | commentCreate_commentCreate_creator_User;

export interface commentCreate_commentCreate_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface commentCreate_commentCreate {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: commentCreate_commentCreate_commentedOn;
  creator: commentCreate_commentCreate_creator | null;
  translations: commentCreate_commentCreate_translations[];
}

export interface commentCreate {
  commentCreate: commentCreate_commentCreate;
}

export interface commentCreateVariables {
  input: CommentCreateInput;
}
