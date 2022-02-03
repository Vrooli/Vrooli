/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceAddInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceAdd
// ====================================================

export interface resourceAdd_resourceAdd {
  __typename: "Resource";
  id: string;
  title: string;
  description: string | null;
  link: string;
  usedFor: ResourceUsedFor | null;
}

export interface resourceAdd {
  resourceAdd: resourceAdd_resourceAdd;
}

export interface resourceAddVariables {
  input: ResourceAddInput;
}
