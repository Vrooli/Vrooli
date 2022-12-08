/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: projectFields
// ====================================================

export interface projectFields_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface projectFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface projectFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface projectFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projectFields_resourceLists_resources_translations[];
}

export interface projectFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: projectFields_resourceLists_translations[];
  resources: projectFields_resourceLists_resources[];
}

export interface projectFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectFields_tags {
  __typename: "Tag";
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

export interface projectFields_owner_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface projectFields_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: projectFields_owner_Organization_translations[];
  permissionsOrganization: projectFields_owner_Organization_permissionsOrganization | null;
}

export interface projectFields_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type projectFields_owner = projectFields_owner_Organization | projectFields_owner_User;

export interface projectFields {
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
  permissionsProject: projectFields_permissionsProject;
  resourceLists: projectFields_resourceLists[] | null;
  tags: projectFields_tags[];
  translations: projectFields_translations[];
  owner: projectFields_owner | null;
}
