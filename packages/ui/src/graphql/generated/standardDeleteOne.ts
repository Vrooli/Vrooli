/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardDeleteOne
// ====================================================

export interface standardDeleteOne_standardDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface standardDeleteOne {
  standardDeleteOne: standardDeleteOne_standardDeleteOne;
}

export interface standardDeleteOneVariables {
  input: DeleteOneInput;
}
