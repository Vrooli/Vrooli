/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationCreateInput, MemberRole, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationCreate
// ====================================================

export interface organizationCreate_organizationCreate_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organizationCreate_organizationCreate_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: organizationCreate_organizationCreate_resources_translations[];
}

export interface organizationCreate_organizationCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationCreate_organizationCreate_tags {
  __typename: "Tag";
  id: string;
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
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  resources: organizationCreate_organizationCreate_resources[];
  tags: organizationCreate_organizationCreate_tags[];
  translations: organizationCreate_organizationCreate_translations[];
}

export interface organizationCreate {
  organizationCreate: organizationCreate_organizationCreate;
}

export interface organizationCreateVariables {
  input: OrganizationCreateInput;
}
