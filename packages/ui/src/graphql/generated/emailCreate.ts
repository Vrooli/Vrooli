/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailCreate
// ====================================================

export interface emailCreate_emailCreate {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface emailCreate {
  emailCreate: emailCreate_emailCreate;
}

export interface emailCreateVariables {
  input: EmailCreateInput;
}
