/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: signUp
// ====================================================

export interface signUp_signUp_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface signUp_signUp_roles {
  __typename: "CustomerRole";
  role: signUp_signUp_roles_role;
}

export interface signUp_signUp {
  __typename: "Customer";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: signUp_signUp_roles[];
}

export interface signUp {
  signUp: signUp_signUp;
}

export interface signUpVariables {
  username: string;
  email: string;
  theme: string;
  marketingEmails: boolean;
  password: string;
}
