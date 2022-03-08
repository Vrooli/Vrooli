/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectUpdateInput, MemberRole, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectUpdate
// ====================================================

export interface projectUpdate_projectUpdate_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projectUpdate_projectUpdate_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projectUpdate_projectUpdate_resources_translations[];
}

export interface projectUpdate_projectUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectUpdate_projectUpdate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: projectUpdate_projectUpdate_tags_translations[];
}

export interface projectUpdate_projectUpdate_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface projectUpdate_projectUpdate_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface projectUpdate_projectUpdate_owner_Organization {
  __typename: "Organization";
  id: string;
  translations: projectUpdate_projectUpdate_owner_Organization_translations[];
}

export interface projectUpdate_projectUpdate_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type projectUpdate_projectUpdate_owner = projectUpdate_projectUpdate_owner_Organization | projectUpdate_projectUpdate_owner_User;

export interface projectUpdate_projectUpdate {
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
  resources: projectUpdate_projectUpdate_resources[] | null;
  tags: projectUpdate_projectUpdate_tags[];
  translations: projectUpdate_projectUpdate_translations[];
  owner: projectUpdate_projectUpdate_owner | null;
}

export interface projectUpdate {
  projectUpdate: projectUpdate_projectUpdate;
}

export interface projectUpdateVariables {
  input: ProjectUpdateInput;
}
