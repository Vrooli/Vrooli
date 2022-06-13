/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentSearchInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: comments
// ====================================================

export interface comments_comments_threads_childThreads_childThreads_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_childThreads_childThreads_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_childThreads_childThreads_comment_commentedOn_Project_translations[];
}

export interface comments_comments_threads_childThreads_childThreads_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface comments_comments_threads_childThreads_childThreads_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: comments_comments_threads_childThreads_childThreads_comment_commentedOn_Routine_translations[];
}

export interface comments_comments_threads_childThreads_childThreads_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type comments_comments_threads_childThreads_childThreads_comment_commentedOn = comments_comments_threads_childThreads_childThreads_comment_commentedOn_Project | comments_comments_threads_childThreads_childThreads_comment_commentedOn_Routine | comments_comments_threads_childThreads_childThreads_comment_commentedOn_Standard;

export interface comments_comments_threads_childThreads_childThreads_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_childThreads_childThreads_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_childThreads_childThreads_comment_creator_Organization_translations[];
}

export interface comments_comments_threads_childThreads_childThreads_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type comments_comments_threads_childThreads_childThreads_comment_creator = comments_comments_threads_childThreads_childThreads_comment_creator_Organization | comments_comments_threads_childThreads_childThreads_comment_creator_User;

export interface comments_comments_threads_childThreads_childThreads_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface comments_comments_threads_childThreads_childThreads_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: comments_comments_threads_childThreads_childThreads_comment_commentedOn;
  creator: comments_comments_threads_childThreads_childThreads_comment_creator | null;
  translations: comments_comments_threads_childThreads_childThreads_comment_translations[];
}

export interface comments_comments_threads_childThreads_childThreads {
  __typename: "CommentThread";
  comment: comments_comments_threads_childThreads_childThreads_comment;
  endCursor: string | null;
  totalInThread: number | null;
}

export interface comments_comments_threads_childThreads_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_childThreads_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_childThreads_comment_commentedOn_Project_translations[];
}

export interface comments_comments_threads_childThreads_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface comments_comments_threads_childThreads_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: comments_comments_threads_childThreads_comment_commentedOn_Routine_translations[];
}

export interface comments_comments_threads_childThreads_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type comments_comments_threads_childThreads_comment_commentedOn = comments_comments_threads_childThreads_comment_commentedOn_Project | comments_comments_threads_childThreads_comment_commentedOn_Routine | comments_comments_threads_childThreads_comment_commentedOn_Standard;

export interface comments_comments_threads_childThreads_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_childThreads_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_childThreads_comment_creator_Organization_translations[];
}

export interface comments_comments_threads_childThreads_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type comments_comments_threads_childThreads_comment_creator = comments_comments_threads_childThreads_comment_creator_Organization | comments_comments_threads_childThreads_comment_creator_User;

export interface comments_comments_threads_childThreads_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface comments_comments_threads_childThreads_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: comments_comments_threads_childThreads_comment_commentedOn;
  creator: comments_comments_threads_childThreads_comment_creator | null;
  translations: comments_comments_threads_childThreads_comment_translations[];
}

export interface comments_comments_threads_childThreads {
  __typename: "CommentThread";
  childThreads: comments_comments_threads_childThreads_childThreads[];
  comment: comments_comments_threads_childThreads_comment;
  endCursor: string | null;
  totalInThread: number | null;
}

export interface comments_comments_threads_comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_comment_commentedOn_Project_translations[];
}

export interface comments_comments_threads_comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface comments_comments_threads_comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: comments_comments_threads_comment_commentedOn_Routine_translations[];
}

export interface comments_comments_threads_comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type comments_comments_threads_comment_commentedOn = comments_comments_threads_comment_commentedOn_Project | comments_comments_threads_comment_commentedOn_Routine | comments_comments_threads_comment_commentedOn_Standard;

export interface comments_comments_threads_comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface comments_comments_threads_comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: comments_comments_threads_comment_creator_Organization_translations[];
}

export interface comments_comments_threads_comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type comments_comments_threads_comment_creator = comments_comments_threads_comment_creator_Organization | comments_comments_threads_comment_creator_User;

export interface comments_comments_threads_comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface comments_comments_threads_comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: comments_comments_threads_comment_commentedOn;
  creator: comments_comments_threads_comment_creator | null;
  translations: comments_comments_threads_comment_translations[];
}

export interface comments_comments_threads {
  __typename: "CommentThread";
  childThreads: comments_comments_threads_childThreads[];
  comment: comments_comments_threads_comment;
  endCursor: string | null;
  totalInThread: number | null;
}

export interface comments_comments {
  __typename: "CommentSearchResult";
  endCursor: string | null;
  totalThreads: number | null;
  threads: comments_comments_threads[] | null;
}

export interface comments {
  comments: comments_comments;
}

export interface commentsVariables {
  input: CommentSearchInput;
}
