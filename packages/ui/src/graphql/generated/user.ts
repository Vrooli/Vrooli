/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: user
// ====================================================

export interface user_user {
  __typename: "User";
  id: string;
  username: string | null;
}

export interface user {
  user: user_user | null;
}

export interface userVariables {
  input: FindByIdInput;
}
