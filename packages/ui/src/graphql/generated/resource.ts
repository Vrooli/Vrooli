/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: resource
// ====================================================

export interface resource_resource_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface resource_resource {
  __typename: "Resource";
  id: string;
  link: string;
  usedFor: ResourceUsedFor | null;
  translations: resource_resource_translations[];
}

export interface resource {
  resource: resource_resource | null;
}

export interface resourceVariables {
  input: FindByIdInput;
}
