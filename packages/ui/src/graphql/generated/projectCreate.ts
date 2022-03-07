/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectCreate
// ====================================================

export interface projectCreate_projectCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectCreate_projectCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: projectCreate_projectCreate_tags_translations[];
}

export interface projectCreate_projectCreate_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface projectCreate_projectCreate_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface projectCreate_projectCreate_owner_Organization {
  __typename: "Organization";
  id: string;
  translations: projectCreate_projectCreate_owner_Organization_translations[];
}

export interface projectCreate_projectCreate_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectCreate_projectCreate_owner = projectCreate_projectCreate_owner_Organization | projectCreate_projectCreate_owner_User;

export interface projectCreate_projectCreate {
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
  tags: projectCreate_projectCreate_tags[];
  translations: projectCreate_projectCreate_translations[];
  owner: projectCreate_projectCreate_owner | null;
}

export interface projectCreate {
  projectCreate: projectCreate_projectCreate;
}

export interface projectCreateVariables {
  input: ProjectCreateInput;
}
