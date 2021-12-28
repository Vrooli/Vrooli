/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteManyInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardDeleteMany
// ====================================================

export interface standardDeleteMany_standardDeleteMany {
  __typename: "Count";
  count: number | null;
}

export interface standardDeleteMany {
  standardDeleteMany: standardDeleteMany_standardDeleteMany;
}

export interface standardDeleteManyVariables {
  input: DeleteManyInput;
}
