/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: resourceFields
// ====================================================

export interface resourceFields_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface resourceFields {
  __typename: "Resource";
  id: string;
  index: number | null;
  link: string;
  usedFor: ResourceUsedFor | null;
  translations: resourceFields_translations[];
}
