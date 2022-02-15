/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentDeleteOne
// ====================================================

export interface commentDeleteOne_commentDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface commentDeleteOne {
  commentDeleteOne: commentDeleteOne_commentDeleteOne;
}

export interface commentDeleteOneVariables {
  input: DeleteOneInput;
}
