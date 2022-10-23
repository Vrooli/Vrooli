/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailLogInInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailLogIn
// ====================================================

export interface emailLogIn_emailLogIn_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: (string | null)[] | null;
  name: string | null;
  theme: string | null;
}

export interface emailLogIn_emailLogIn {
  __typename: "Session";
  isLoggedIn: boolean;
  users: emailLogIn_emailLogIn_users[] | null;
}

export interface emailLogIn {
  emailLogIn: emailLogIn_emailLogIn;
}

export interface emailLogInVariables {
  input: EmailLogInInput;
}
