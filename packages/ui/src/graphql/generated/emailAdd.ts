/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailAdd
// ====================================================

export interface emailAdd_emailAdd {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface emailAdd {
  emailAdd: emailAdd_emailAdd;
}

export interface emailAddVariables {
  input: EmailInput;
}
