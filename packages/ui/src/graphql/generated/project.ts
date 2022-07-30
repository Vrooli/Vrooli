/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdOrHandleInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: project
// ====================================================

export interface project_project_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface project_project_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface project_project_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface project_project_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: project_project_resourceLists_resources_translations[];
}

export interface project_project_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: project_project_resourceLists_translations[];
  resources: project_project_resourceLists_resources[];
}

export interface project_project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface project_project_tags {
  __typename: "Tag";
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
  handle: string | null;
  translations: project_project_owner_Organization_translations[];
}

export interface project_project_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type project_project_owner = project_project_owner_Organization | project_project_owner_User;

export interface project_project {
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
  permissionsProject: project_project_permissionsProject;
  resourceLists: project_project_resourceLists[] | null;
  tags: project_project_tags[];
  translations: project_project_translations[];
  owner: project_project_owner | null;
}

export interface project {
  project: project_project | null;
}

export interface projectVariables {
  input: FindByIdOrHandleInput;
}
