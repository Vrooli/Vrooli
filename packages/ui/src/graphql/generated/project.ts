/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: project
// ====================================================

export interface project_project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface project_project_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: project_project_tags_translations[];
}

export interface project_project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface project_project_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface project_project_owner_Organization {
  __typename: "Organization";
  id: string;
  translations: project_project_owner_Organization_translations[];
}

export interface project_project_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type project_project_owner = project_project_owner_Organization | project_project_owner_User;

export interface project_project {
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
  tags: project_project_tags[];
  translations: project_project_translations[];
  owner: project_project_owner | null;
}

export interface project {
  project: project_project | null;
}

export interface projectVariables {
  input: FindByIdInput;
}
