/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ResetPasswordInput, AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: resetPassword
// ====================================================

export interface emailResetPassword_resetPassword_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface emailResetPassword_resetPassword_roles {
  __typename: "UserRole";
  role: emailResetPassword_resetPassword_roles_role;
}

export interface emailResetPassword_resetPassword {
  __typename: "User";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: emailResetPassword_resetPassword_roles[];
}

export interface emailResetPassword {
  resetPassword: emailResetPassword_resetPassword;
}

export interface emailResetPasswordVariables {
  input: ResetPasswordInput;
}
