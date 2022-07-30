/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectCreateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectCreate
// ====================================================

export interface projectCreate_projectCreate_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface projectCreate_projectCreate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projectCreate_projectCreate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projectCreate_projectCreate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projectCreate_projectCreate_resourceLists_resources_translations[];
}

export interface projectCreate_projectCreate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: projectCreate_projectCreate_resourceLists_translations[];
  resources: projectCreate_projectCreate_resourceLists_resources[];
}

export interface projectCreate_projectCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectCreate_projectCreate_tags {
  __typename: "Tag";
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
  handle: string | null;
  translations: projectCreate_projectCreate_owner_Organization_translations[];
}

export interface projectCreate_projectCreate_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type projectCreate_projectCreate_owner = projectCreate_projectCreate_owner_Organization | projectCreate_projectCreate_owner_User;

export interface projectCreate_projectCreate {
  __typename: "Project";
  id: string;
  completedAt: any | null;
  created_at: any;
  handle: string | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  score: number;
  stars: number;
  permissionsProject: projectCreate_projectCreate_permissionsProject;
  resourceLists: projectCreate_projectCreate_resourceLists[] | null;
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
