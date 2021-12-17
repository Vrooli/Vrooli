/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { SignUpInput, AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: signUp
// ====================================================

export interface signUp_signUp_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface signUp_signUp_roles {
  __typename: "UserRole";
  role: signUp_signUp_roles_role;
}

export interface signUp_signUp {
  __typename: "User";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: signUp_signUp_roles[];
}

export interface signUp {
  signUp: signUp_signUp;
}

export interface signUpVariables {
  input: SignUpInput;
}
