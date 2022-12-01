/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailResetPasswordInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailResetPassword
// ====================================================

export interface emailResetPassword_emailResetPassword_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: string[];
  name: string | null;
  theme: string | null;
}

export interface emailResetPassword_emailResetPassword {
  __typename: "Session";
  isLoggedIn: boolean;
  users: emailResetPassword_emailResetPassword_users[] | null;
}

export interface emailResetPassword {
  emailResetPassword: emailResetPassword_emailResetPassword;
}

export interface emailResetPasswordVariables {
  input: EmailResetPasswordInput;
}
