/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceUpdate
// ====================================================

export interface resourceUpdate_resourceUpdate {
  __typename: "Resource";
  id: string;
  name: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface resourceUpdate {
  resourceUpdate: resourceUpdate_resourceUpdate;
}

export interface resourceUpdateVariables {
  input: ResourceInput;
}
