/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceCreateInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceCreate
// ====================================================

export interface resourceCreate_resourceCreate_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface resourceCreate_resourceCreate {
  __typename: "Resource";
  id: string;
  link: string;
  usedFor: ResourceUsedFor | null;
  translations: resourceCreate_resourceCreate_translations[];
}

export interface resourceCreate {
  resourceCreate: resourceCreate_resourceCreate;
}

export interface resourceCreateVariables {
  input: ResourceCreateInput;
}
