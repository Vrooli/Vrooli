/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineDeleteOne
// ====================================================

export interface routineDeleteOne_routineDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface routineDeleteOne {
  routineDeleteOne: routineDeleteOne_routineDeleteOne;
}

export interface routineDeleteOneVariables {
  input: DeleteOneInput;
}
