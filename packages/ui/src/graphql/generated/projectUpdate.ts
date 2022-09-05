/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectUpdateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectUpdate
// ====================================================

export interface projectUpdate_projectUpdate_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface projectUpdate_projectUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projectUpdate_projectUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface projectUpdate_projectUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projectUpdate_projectUpdate_resourceLists_resources_translations[];
}

export interface projectUpdate_projectUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: projectUpdate_projectUpdate_resourceLists_translations[];
  resources: projectUpdate_projectUpdate_resourceLists_resources[];
}

export interface projectUpdate_projectUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectUpdate_projectUpdate_tags {
  __typename: "Tag";
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

export interface projectUpdate_projectUpdate_owner_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface projectUpdate_projectUpdate_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: projectUpdate_projectUpdate_owner_Organization_translations[];
  permissionsOrganization: projectUpdate_projectUpdate_owner_Organization_permissionsOrganization | null;
}

export interface projectUpdate_projectUpdate_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type projectUpdate_projectUpdate_owner = projectUpdate_projectUpdate_owner_Organization | projectUpdate_projectUpdate_owner_User;

export interface projectUpdate_projectUpdate {
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
  permissionsProject: projectUpdate_projectUpdate_permissionsProject;
  resourceLists: projectUpdate_projectUpdate_resourceLists[] | null;
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
