/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceUpdateInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceUpdate
// ====================================================

export interface resourceUpdate_resourceUpdate {
  __typename: "Resource";
  id: string;
  title: string;
  description: string | null;
  link: string;
  usedFor: ResourceUsedFor | null;
}

export interface resourceUpdate {
  resourceUpdate: resourceUpdate_resourceUpdate;
}

export interface resourceUpdateVariables {
  input: ResourceUpdateInput;
}
