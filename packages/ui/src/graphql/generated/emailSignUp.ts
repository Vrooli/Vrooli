/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailSignUpInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailSignUp
// ====================================================

export interface emailSignUp_emailSignUp_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: (string | null)[] | null;
  name: string | null;
  theme: string | null;
}

export interface emailSignUp_emailSignUp {
  __typename: "Session";
  isLoggedIn: boolean;
  users: emailSignUp_emailSignUp_users[] | null;
}

export interface emailSignUp {
  emailSignUp: emailSignUp_emailSignUp;
}

export interface emailSignUpVariables {
  input: EmailSignUpInput;
}
