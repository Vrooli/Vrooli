/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: organizationFields
// ====================================================

export interface organizationFields_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface organizationFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: organizationFields_resourceLists_resources_translations[];
}

export interface organizationFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: organizationFields_resourceLists_translations[];
  resources: organizationFields_resourceLists_resources[];
}

export interface organizationFields_roles_translations {
  __typename: "RoleTranslation";
  id: string;
  language: string;
  description: string;
}

export interface organizationFields_roles {
  __typename: "Role";
  id: string;
  created_at: any;
  updated_at: any;
  title: string;
  translations: organizationFields_roles_translations[];
}

export interface organizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationFields_tags {
  __typename: "Tag";
  tag: string;
  translations: organizationFields_tags_translations[];
}

export interface organizationFields_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organizationFields {
  __typename: "Organization";
  id: string;
  created_at: any;
  handle: string | null;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  stars: number;
  permissionsOrganization: organizationFields_permissionsOrganization | null;
  resourceLists: organizationFields_resourceLists[];
  roles: organizationFields_roles[] | null;
  tags: organizationFields_tags[];
  translations: organizationFields_translations[];
}
