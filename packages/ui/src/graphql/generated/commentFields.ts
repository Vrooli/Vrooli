/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: commentFields
// ====================================================

export interface commentFields_creator_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface commentFields_creator_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type commentFields_creator = commentFields_creator_Organization | commentFields_creator_User;

export interface commentFields_commentedOn_Project {
  __typename: "Project";
  id: string;
  name: string;
}

export interface commentFields_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface commentFields_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type commentFields_commentedOn = commentFields_commentedOn_Project | commentFields_commentedOn_Routine | commentFields_commentedOn_Standard;

export interface commentFields {
  __typename: "Comment";
  id: string;
  text: string | null;
  created_at: any;
  updated_at: any;
  score: number | null;
  isUpvoted: boolean | null;
  isStarred: boolean | null;
  creator: commentFields_creator | null;
  commentedOn: commentFields_commentedOn;
}
