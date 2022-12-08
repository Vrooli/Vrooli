/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: projectResourceListFields
// ====================================================

export interface projectResourceListFields_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface projectResourceListFields_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface projectResourceListFields_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: projectResourceListFields_resources_translations[];
}

export interface projectResourceListFields {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: projectResourceListFields_translations[];
  resources: projectResourceListFields_resources[];
}
