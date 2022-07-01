/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: standard
// ====================================================

export interface standard_standard_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface standard_standard_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface standard_standard_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: standard_standard_resourceLists_resources_translations[];
}

export interface standard_standard_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: standard_standard_resourceLists_translations[];
  resources: standard_standard_resourceLists_resources[];
}

export interface standard_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standard_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: standard_standard_tags_translations[];
}

export interface standard_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standard_standard_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface standard_standard_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: standard_standard_creator_Organization_translations[];
}

export interface standard_standard_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type standard_standard_creator = standard_standard_creator_Organization | standard_standard_creator_User;

export interface standard_standard {
  __typename: "Standard";
  id: string;
  isInternal: boolean;
  name: string;
  role: MemberRole | null;
  type: string;
  props: string;
  yup: string | null;
  default: string | null;
  created_at: any;
  resourceLists: standard_standard_resourceLists[];
  tags: standard_standard_tags[];
  translations: standard_standard_translations[];
  creator: standard_standard_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
  version: string;
}

export interface standard {
  standard: standard_standard | null;
}

export interface standardVariables {
  input: FindByIdInput;
}
