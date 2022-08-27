/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteManyInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resourceDeleteMany
// ====================================================

export interface resourceDeleteMany_resourceDeleteMany {
  __typename: "Count";
  count: number;
}

export interface resourceDeleteMany {
  resourceDeleteMany: resourceDeleteMany_resourceDeleteMany;
}

export interface resourceDeleteManyVariables {
  input: DeleteManyInput;
}
