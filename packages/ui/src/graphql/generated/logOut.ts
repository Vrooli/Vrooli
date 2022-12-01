/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { LogOutInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: logOut
// ====================================================

export interface logOut_logOut_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: string[];
  name: string | null;
  theme: string | null;
}

export interface logOut_logOut {
  __typename: "Session";
  isLoggedIn: boolean;
  users: logOut_logOut_users[] | null;
}

export interface logOut {
  logOut: logOut_logOut;
}

export interface logOutVariables {
  input: LogOutInput;
}
