/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: projectDeleteOne
// ====================================================

export interface projectDeleteOne_projectDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface projectDeleteOne {
  projectDeleteOne: projectDeleteOne_projectDeleteOne;
}

export interface projectDeleteOneVariables {
  input: DeleteOneInput;
}
