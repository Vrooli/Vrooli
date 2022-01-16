/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: voteDeleteOne
// ====================================================

export interface voteDeleteOne_voteDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface voteDeleteOne {
  voteDeleteOne: voteDeleteOne_voteDeleteOne;
}

export interface voteDeleteOneVariables {
  input: DeleteOneInput;
}
