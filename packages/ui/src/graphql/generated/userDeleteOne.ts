/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { UserDeleteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: userDeleteOne
// ====================================================

export interface userDeleteOne_userDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface userDeleteOne {
  userDeleteOne: userDeleteOne_userDeleteOne;
}

export interface userDeleteOneVariables {
  input: UserDeleteInput;
}
