/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { SwitchCurrentAccountInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: switchCurrentAccount
// ====================================================

export interface switchCurrentAccount_switchCurrentAccount_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: string[];
  name: string | null;
  theme: string | null;
}

export interface switchCurrentAccount_switchCurrentAccount {
  __typename: "Session";
  isLoggedIn: boolean;
  users: switchCurrentAccount_switchCurrentAccount_users[] | null;
}

export interface switchCurrentAccount {
  switchCurrentAccount: switchCurrentAccount_switchCurrentAccount;
}

export interface switchCurrentAccountVariables {
  input: SwitchCurrentAccountInput;
}
