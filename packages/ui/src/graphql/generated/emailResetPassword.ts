/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailResetPasswordInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailResetPassword
// ====================================================

export interface emailResetPassword_emailResetPassword_roles {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface emailResetPassword_emailResetPassword {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: emailResetPassword_emailResetPassword_roles[];
}

export interface emailResetPassword {
  emailResetPassword: emailResetPassword_emailResetPassword;
}

export interface emailResetPasswordVariables {
  input: EmailResetPasswordInput;
}
