/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailLogInInput, AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: logIn
// ====================================================

export interface logIn_logIn_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface logIn_logIn_roles {
  __typename: "UserRole";
  role: logIn_logIn_roles_role;
}

export interface logIn_logIn {
  __typename: "User";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: logIn_logIn_roles[];
}

export interface emailLogIn {
  logIn: logIn_logIn;
}

export interface logInVariables {
  input: EmailLogInInput;
}
