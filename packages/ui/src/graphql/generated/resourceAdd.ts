/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceAdd
// ====================================================

export interface resourceAdd_resourceAdd {
  __typename: "Resource";
  id: string;
  name: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface resourceAdd {
  resourceAdd: resourceAdd_resourceAdd;
}

export interface resourceAddVariables {
  input: ResourceInput;
}
