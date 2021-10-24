/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resetPassword
// ====================================================

export interface resetPassword_resetPassword_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface resetPassword_resetPassword_roles {
  __typename: "CustomerRole";
  role: resetPassword_resetPassword_roles_role;
}

export interface resetPassword_resetPassword {
  __typename: "Customer";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: resetPassword_resetPassword_roles[];
}

export interface resetPassword {
  resetPassword: resetPassword_resetPassword;
}

export interface resetPasswordVariables {
  id: string;
  code: string;
  newPassword: string;
}
