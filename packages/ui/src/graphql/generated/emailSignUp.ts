/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailSignUpInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailSignUp
// ====================================================

export interface emailSignUp_emailSignUp {
  __typename: "Session";
  id: string | null;
  theme: string;
  isLoggedIn: boolean;
  languages: string[] | null;
}

export interface emailSignUp {
  emailSignUp: emailSignUp_emailSignUp;
}

export interface emailSignUpVariables {
  input: EmailSignUpInput;
}
