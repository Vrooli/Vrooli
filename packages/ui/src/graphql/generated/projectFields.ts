/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: projectFields
// ====================================================

export interface projectFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: projectFields_tags_translations[];
}

export interface projectFields_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface projectFields_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface projectFields_owner_Organization {
  __typename: "Organization";
  id: string;
  translations: projectFields_owner_Organization_translations[];
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
  completedAt: any | null;
  created_at: any;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  tags: projectFields_tags[];
  translations: projectFields_translations[];
  owner: projectFields_owner | null;
}
