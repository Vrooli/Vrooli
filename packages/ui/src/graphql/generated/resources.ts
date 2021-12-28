/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResourcesQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: resources
// ====================================================

export interface resources_resources {
  __typename: "Resource";
  id: string;
  name: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface resources {
  resources: resources_resources[];
}

export interface resourcesVariables {
  input: ResourcesQueryInput;
}
