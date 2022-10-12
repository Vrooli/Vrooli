/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardUpdateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardUpdate
// ====================================================

export interface standardUpdate_standardUpdate_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface standardUpdate_standardUpdate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface standardUpdate_standardUpdate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface standardUpdate_standardUpdate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: standardUpdate_standardUpdate_resourceLists_resources_translations[];
}

export interface standardUpdate_standardUpdate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: standardUpdate_standardUpdate_resourceLists_translations[];
  resources: standardUpdate_standardUpdate_resourceLists_resources[];
}

export interface standardUpdate_standardUpdate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardUpdate_standardUpdate_tags {
  __typename: "Tag";
  tag: string;
  translations: standardUpdate_standardUpdate_tags_translations[];
}

export interface standardUpdate_standardUpdate_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface standardUpdate_standardUpdate_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface standardUpdate_standardUpdate_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: standardUpdate_standardUpdate_creator_Organization_translations[];
}

export interface standardUpdate_standardUpdate_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type standardUpdate_standardUpdate_creator = standardUpdate_standardUpdate_creator_Organization | standardUpdate_standardUpdate_creator_User;

export interface standardUpdate_standardUpdate {
  __typename: "Standard";
  id: string;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  default: string | null;
  created_at: any;
  permissionsStandard: standardUpdate_standardUpdate_permissionsStandard;
  resourceLists: standardUpdate_standardUpdate_resourceLists[];
  tags: standardUpdate_standardUpdate_tags[];
  translations: standardUpdate_standardUpdate_translations[];
  creator: standardUpdate_standardUpdate_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
  version: string;
  versionGroupId: string;
  versions: string[];
}

export interface standardUpdate {
  standardUpdate: standardUpdate_standardUpdate;
}

export interface standardUpdateVariables {
  input: StandardUpdateInput;
}
