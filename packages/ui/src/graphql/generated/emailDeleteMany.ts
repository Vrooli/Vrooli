/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteManyInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailDeleteMany
// ====================================================

export interface emailDeleteMany_emailDeleteMany {
  __typename: "Count";
  count: number | null;
}

export interface emailDeleteMany {
  emailDeleteMany: emailDeleteMany_emailDeleteMany;
}

export interface emailDeleteManyVariables {
  input: DeleteManyInput;
}
