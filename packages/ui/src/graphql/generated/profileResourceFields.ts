/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: profileResourceFields
// ====================================================

export interface profileResourceFields_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface profileResourceFields {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: profileResourceFields_translations[];
}
