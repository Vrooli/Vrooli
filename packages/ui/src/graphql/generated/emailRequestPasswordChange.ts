/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailRequestPasswordChangeInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailRequestPasswordChange
// ====================================================

export interface emailRequestPasswordChange_emailRequestPasswordChange {
  __typename: "Success";
  success: boolean | null;
}

export interface emailRequestPasswordChange {
  emailRequestPasswordChange: emailRequestPasswordChange_emailRequestPasswordChange;
}

export interface emailRequestPasswordChangeVariables {
  input: EmailRequestPasswordChangeInput;
}
