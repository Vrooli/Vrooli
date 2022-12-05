/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteManyInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: deleteMany
// ====================================================

export interface deleteMany_deleteMany {
  __typename: "Count";
  count: number;
}

export interface deleteMany {
  deleteMany: deleteMany_deleteMany;
}

export interface deleteManyVariables {
  input: DeleteManyInput;
}
