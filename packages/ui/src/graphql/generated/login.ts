/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: login
// ====================================================

export interface login_login_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface login_login_roles {
  __typename: "UserRole";
  role: login_login_roles_role;
}

export interface login_login {
  __typename: "User";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: login_login_roles[];
}

export interface login {
  login: login_login;
}

export interface loginVariables {
  email?: string | null;
  password?: string | null;
}
