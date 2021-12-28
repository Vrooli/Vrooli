/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeDeleteOne
// ====================================================

export interface nodeDeleteOne_nodeDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface nodeDeleteOne {
  nodeDeleteOne: nodeDeleteOne_nodeDeleteOne;
}

export interface nodeDeleteOneVariables {
  input: DeleteOneInput;
}
