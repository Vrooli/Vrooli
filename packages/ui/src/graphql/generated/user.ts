/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdOrHandleInput, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: user
// ====================================================

export interface user_user_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface user_user_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface user_user_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: user_user_resourceLists_resources_translations[];
}

export interface user_user_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: user_user_resourceLists_translations[];
  resources: user_user_resourceLists_resources[];
}

export interface user_user_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface user_user {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  reportsCount: number;
  resourceLists: user_user_resourceLists[];
  translations: user_user_translations[];
}

export interface user {
  user: user_user | null;
}

export interface userVariables {
  input: FindByIdOrHandleInput;
}
