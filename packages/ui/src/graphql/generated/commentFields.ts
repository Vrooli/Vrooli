/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: commentFields
// ====================================================

export interface commentFields_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentFields_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: commentFields_commentedOn_Project_translations[];
}

export interface commentFields_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface commentFields_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  translations: commentFields_commentedOn_Routine_translations[];
}

export interface commentFields_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
  type: string;
}

export type commentFields_commentedOn = commentFields_commentedOn_Project | commentFields_commentedOn_Routine | commentFields_commentedOn_Standard;

export interface commentFields_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface commentFields_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: commentFields_creator_Organization_translations[];
}

export interface commentFields_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type commentFields_creator = commentFields_creator_Organization | commentFields_creator_User;

export interface commentFields_permissionsComment {
  __typename: "CommentPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReply: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface commentFields_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface commentFields {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  commentedOn: commentFields_commentedOn;
  creator: commentFields_creator | null;
  permissionsComment: commentFields_permissionsComment | null;
  translations: commentFields_translations[];
}
