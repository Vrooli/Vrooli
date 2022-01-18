/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: projectFields
// ====================================================

export interface projectFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface projectFields_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface projectFields_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectFields_owner = projectFields_owner_Organization | projectFields_owner_User;

export interface projectFields {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
  tags: projectFields_tags[];
  owner: projectFields_owner | null;
  stars: number;
  score: number;
  isUpvoted: boolean;
}
