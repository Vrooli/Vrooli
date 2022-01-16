/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: reportDeleteOne
// ====================================================

export interface reportDeleteOne_reportDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface reportDeleteOne {
  reportDeleteOne: reportDeleteOne_reportDeleteOne;
}

export interface reportDeleteOneVariables {
  input: DeleteOneInput;
}
