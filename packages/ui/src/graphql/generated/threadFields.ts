/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: threadFields
// ====================================================

export interface threadFields_childThreads_childThreads_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_childThreads_childThreads_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: threadFields_childThreads_childThreads_comment_commentedOn_Project_translations[];
}

export interface threadFields_childThreads_childThreads_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface threadFields_childThreads_childThreads_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: threadFields_childThreads_childThreads_comment_commentedOn_Routine_translations[];
}

export interface threadFields_childThreads_childThreads_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type threadFields_childThreads_childThreads_comment_commentedOn = threadFields_childThreads_childThreads_comment_commentedOn_Project | threadFields_childThreads_childThreads_comment_commentedOn_Routine | threadFields_childThreads_childThreads_comment_commentedOn_Standard;

export interface threadFields_childThreads_childThreads_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_childThreads_childThreads_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: threadFields_childThreads_childThreads_comment_creator_Organization_translations[];
}

export interface threadFields_childThreads_childThreads_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type threadFields_childThreads_childThreads_comment_creator = threadFields_childThreads_childThreads_comment_creator_Organization | threadFields_childThreads_childThreads_comment_creator_User;

export interface threadFields_childThreads_childThreads_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface threadFields_childThreads_childThreads_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: threadFields_childThreads_childThreads_comment_commentedOn;
  creator: threadFields_childThreads_childThreads_comment_creator | null;
  translations: threadFields_childThreads_childThreads_comment_translations[];
}

export interface threadFields_childThreads_childThreads {
  __typename: "CommentThread";
  comment: threadFields_childThreads_childThreads_comment;
  endCursor: string | null;
  totalInThread: number | null;
}

export interface threadFields_childThreads_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_childThreads_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: threadFields_childThreads_comment_commentedOn_Project_translations[];
}

export interface threadFields_childThreads_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface threadFields_childThreads_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: threadFields_childThreads_comment_commentedOn_Routine_translations[];
}

export interface threadFields_childThreads_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type threadFields_childThreads_comment_commentedOn = threadFields_childThreads_comment_commentedOn_Project | threadFields_childThreads_comment_commentedOn_Routine | threadFields_childThreads_comment_commentedOn_Standard;

export interface threadFields_childThreads_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_childThreads_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: threadFields_childThreads_comment_creator_Organization_translations[];
}

export interface threadFields_childThreads_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type threadFields_childThreads_comment_creator = threadFields_childThreads_comment_creator_Organization | threadFields_childThreads_comment_creator_User;

export interface threadFields_childThreads_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface threadFields_childThreads_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: threadFields_childThreads_comment_commentedOn;
  creator: threadFields_childThreads_comment_creator | null;
  translations: threadFields_childThreads_comment_translations[];
}

export interface threadFields_childThreads {
  __typename: "CommentThread";
  childThreads: threadFields_childThreads_childThreads[];
  comment: threadFields_childThreads_comment;
  endCursor: string | null;
  totalInThread: number | null;
}

export interface threadFields_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: threadFields_comment_commentedOn_Project_translations[];
}

export interface threadFields_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface threadFields_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: threadFields_comment_commentedOn_Routine_translations[];
}

export interface threadFields_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type threadFields_comment_commentedOn = threadFields_comment_commentedOn_Project | threadFields_comment_commentedOn_Routine | threadFields_comment_commentedOn_Standard;

export interface threadFields_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadFields_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: threadFields_comment_creator_Organization_translations[];
}

export interface threadFields_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type threadFields_comment_creator = threadFields_comment_creator_Organization | threadFields_comment_creator_User;

export interface threadFields_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface threadFields_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: threadFields_comment_commentedOn;
  creator: threadFields_comment_creator | null;
  translations: threadFields_comment_translations[];
}

export interface threadFields {
  __typename: "CommentThread";
  childThreads: threadFields_childThreads[];
  comment: threadFields_comment;
  endCursor: string | null;
  totalInThread: number | null;
}
