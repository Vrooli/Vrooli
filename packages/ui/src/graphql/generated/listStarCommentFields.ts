/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarCommentFields
// ====================================================

export interface listStarCommentFields_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listStarCommentFields_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: listStarCommentFields_commentedOn_Project_translations[];
}

export interface listStarCommentFields_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name string;
}

export interface listStarCommentFields_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: listStarCommentFields_commentedOn_Routine_translations[];
}

export interface listStarCommentFields_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type listStarCommentFields_commentedOn = listStarCommentFields_commentedOn_Project | listStarCommentFields_commentedOn_Routine | listStarCommentFields_commentedOn_Standard;

export interface listStarCommentFields_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface listStarCommentFields_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: listStarCommentFields_creator_Organization_translations[];
}

export interface listStarCommentFields_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type listStarCommentFields_creator = listStarCommentFields_creator_Organization | listStarCommentFields_creator_User;

export interface listStarCommentFields_permissionsComment {
  __typename: "CommentPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReply: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarCommentFields_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface listStarCommentFields {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  commentedOn: listStarCommentFields_commentedOn;
  creator: listStarCommentFields_creator | null;
  permissionsComment: listStarCommentFields_permissionsComment | null;
  translations: listStarCommentFields_translations[];
}
