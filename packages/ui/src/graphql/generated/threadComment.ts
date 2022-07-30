/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: threadComment
// ====================================================

export interface threadComment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadComment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: threadComment_commentedOn_Project_translations[];
}

export interface threadComment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface threadComment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  translations: threadComment_commentedOn_Routine_translations[];
}

export interface threadComment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
  type: string;
}

export type threadComment_commentedOn = threadComment_commentedOn_Project | threadComment_commentedOn_Routine | threadComment_commentedOn_Standard;

export interface threadComment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface threadComment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: threadComment_creator_Organization_translations[];
}

export interface threadComment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type threadComment_creator = threadComment_creator_Organization | threadComment_creator_User;

export interface threadComment_permissionsComment {
  __typename: "CommentPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface threadComment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface threadComment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  commentedOn: threadComment_commentedOn;
  creator: threadComment_creator | null;
  permissionsComment: threadComment_permissionsComment | null;
  translations: threadComment_translations[];
}
