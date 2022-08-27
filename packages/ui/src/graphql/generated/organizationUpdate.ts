/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationUpdateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationUpdate
// ====================================================

export interface organizationUpdate_organizationUpdate_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface organizationUpdate_organizationUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationUpdate_organizationUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationUpdate_organizationUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: organizationUpdate_organizationUpdate_resourceLists_resources_translations[];
}

export interface organizationUpdate_organizationUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: organizationUpdate_organizationUpdate_resourceLists_translations[];
  resources: organizationUpdate_organizationUpdate_resourceLists_resources[];
}

export interface organizationUpdate_organizationUpdate_roles_translations {
  __typename: "RoleTranslation";
  id: string;
  language: string;
  description: string;
}

export interface organizationUpdate_organizationUpdate_roles {
  __typename: "Role";
  id: string;
  created_at: any;
  updated_at: any;
  title: string;
  translations: organizationUpdate_organizationUpdate_roles_translations[];
}

export interface organizationUpdate_organizationUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationUpdate_organizationUpdate_tags {
  __typename: "Tag";
  tag: string;
  translations: organizationUpdate_organizationUpdate_tags_translations[];
}

export interface organizationUpdate_organizationUpdate_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organizationUpdate_organizationUpdate {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  stars: number;
  reportsCount: number;
  permissionsOrganization: organizationUpdate_organizationUpdate_permissionsOrganization | null;
  resourceLists: organizationUpdate_organizationUpdate_resourceLists[];
  roles: organizationUpdate_organizationUpdate_roles[] | null;
  tags: organizationUpdate_organizationUpdate_tags[];
  translations: organizationUpdate_organizationUpdate_translations[];
}

export interface organizationUpdate {
  organizationUpdate: organizationUpdate_organizationUpdate;
}

export interface organizationUpdateVariables {
  input: OrganizationUpdateInput;
}
