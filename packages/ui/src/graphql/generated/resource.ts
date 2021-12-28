/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: resource
// ====================================================

export interface resource_resource {
  __typename: "Resource";
  id: string;
  name: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface resource {
  resource: resource_resource | null;
}

export interface resourceVariables {
  input: FindByIdInput;
}
