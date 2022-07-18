/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationUpdateInput, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationUpdate
// ====================================================

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
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  resourceLists: organizationUpdate_organizationUpdate_resourceLists[];
  tags: organizationUpdate_organizationUpdate_tags[];
  translations: organizationUpdate_organizationUpdate_translations[];
}

export interface organizationUpdate {
  organizationUpdate: organizationUpdate_organizationUpdate;
}

export interface organizationUpdateVariables {
  input: OrganizationUpdateInput;
}
