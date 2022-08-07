/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { SendVerificationEmailInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: sendVerificationEmail
// ====================================================

export interface sendVerificationEmail_sendVerificationEmail {
  __typename: "Success";
  success: boolean;
}

export interface sendVerificationEmail {
  sendVerificationEmail: sendVerificationEmail_sendVerificationEmail;
}

export interface sendVerificationEmailVariables {
  input: SendVerificationEmailInput;
}
