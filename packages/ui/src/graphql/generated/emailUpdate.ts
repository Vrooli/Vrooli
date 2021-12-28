/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailUpdate
// ====================================================

export interface emailUpdate_emailUpdate {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface emailUpdate {
  emailUpdate: emailUpdate_emailUpdate;
}

export interface emailUpdateVariables {
  input: EmailInput;
}
