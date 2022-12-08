/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardCreateInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardCreate
// ====================================================

export interface standardCreate_standardCreate_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface standardCreate_standardCreate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface standardCreate_standardCreate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface standardCreate_standardCreate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: standardCreate_standardCreate_resourceLists_resources_translations[];
}

export interface standardCreate_standardCreate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: standardCreate_standardCreate_resourceLists_translations[];
  resources: standardCreate_standardCreate_resourceLists_resources[];
}

export interface standardCreate_standardCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardCreate_standardCreate_tags {
  __typename: "Tag";
  tag: string;
  translations: standardCreate_standardCreate_tags_translations[];
}

export interface standardCreate_standardCreate_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface standardCreate_standardCreate_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface standardCreate_standardCreate_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: standardCreate_standardCreate_creator_Organization_translations[];
}

export interface standardCreate_standardCreate_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type standardCreate_standardCreate_creator = standardCreate_standardCreate_creator_Organization | standardCreate_standardCreate_creator_User;

export interface standardCreate_standardCreate {
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
  permissionsStandard: standardCreate_standardCreate_permissionsStandard;
  resourceLists: standardCreate_standardCreate_resourceLists[];
  tags: standardCreate_standardCreate_tags[];
  translations: standardCreate_standardCreate_translations[];
  creator: standardCreate_standardCreate_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardCreate {
  standardCreate: standardCreate_standardCreate;
}

export interface standardCreateVariables {
  input: StandardCreateInput;
}
