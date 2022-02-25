/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourceCreateInput, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceCreate
// ====================================================

export interface resourceCreate_resourceCreate {
  __typename: "Resource";
  id: string;
  title: string | null;
  description: string | null;
  link: string;
  usedFor: ResourceUsedFor | null;
}

export interface resourceCreate {
  resourceCreate: resourceCreate_resourceCreate;
}

export interface resourceCreateVariables {
  input: ResourceCreateInput;
}
