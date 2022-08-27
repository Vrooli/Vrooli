/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationCreateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationCreate
// ====================================================

export interface organizationCreate_organizationCreate_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface organizationCreate_organizationCreate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationCreate_organizationCreate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationCreate_organizationCreate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: organizationCreate_organizationCreate_resourceLists_resources_translations[];
}

export interface organizationCreate_organizationCreate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: organizationCreate_organizationCreate_resourceLists_translations[];
  resources: organizationCreate_organizationCreate_resourceLists_resources[];
}

export interface organizationCreate_organizationCreate_roles_translations {
  __typename: "RoleTranslation";
  id: string;
  language: string;
  description: string;
}

export interface organizationCreate_organizationCreate_roles {
  __typename: "Role";
  id: string;
  created_at: any;
  updated_at: any;
  title: string;
  translations: organizationCreate_organizationCreate_roles_translations[];
}

export interface organizationCreate_organizationCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationCreate_organizationCreate_tags {
  __typename: "Tag";
  tag: string;
  translations: organizationCreate_organizationCreate_tags_translations[];
}

export interface organizationCreate_organizationCreate_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organizationCreate_organizationCreate {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  stars: number;
  reportsCount: number;
  permissionsOrganization: organizationCreate_organizationCreate_permissionsOrganization | null;
  resourceLists: organizationCreate_organizationCreate_resourceLists[];
  roles: organizationCreate_organizationCreate_roles[] | null;
  tags: organizationCreate_organizationCreate_tags[];
  translations: organizationCreate_organizationCreate_translations[];
}

export interface organizationCreate {
  organizationCreate: organizationCreate_organizationCreate;
}

export interface organizationCreateVariables {
  input: OrganizationCreateInput;
}
