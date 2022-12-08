/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceUpdateInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceUpdate
// ====================================================

export interface resourceUpdate_resourceUpdate_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface resourceUpdate_resourceUpdate {
  __typename: "Resource";
  id: string;
  index: number | null;
  link: string;
  usedFor: ResourceUsedFor | null;
  translations: resourceUpdate_resourceUpdate_translations[];
}

export interface resourceUpdate {
  resourceUpdate: resourceUpdate_resourceUpdate;
}

export interface resourceUpdateVariables {
  input: ResourceUpdateInput;
}
