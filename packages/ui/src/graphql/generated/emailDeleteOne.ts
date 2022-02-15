/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailDeleteOne
// ====================================================

export interface emailDeleteOne_emailDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface emailDeleteOne {
  emailDeleteOne: emailDeleteOne_emailDeleteOne;
}

export interface emailDeleteOneVariables {
  input: DeleteOneInput;
}
