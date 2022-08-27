/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: deleteOne
// ====================================================

export interface deleteOne_deleteOne {
  __typename: "Success";
  success: boolean;
}

export interface deleteOne {
  deleteOne: deleteOne_deleteOne;
}

export interface deleteOneVariables {
  input: DeleteOneInput;
}
