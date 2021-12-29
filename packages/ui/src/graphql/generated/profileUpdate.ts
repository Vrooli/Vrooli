/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileUpdate
// ====================================================

export interface profileUpdate_profileUpdate {
  __typename: "Profile";
  id: string;
  username: string | null;
}

export interface profileUpdate {
  profileUpdate: profileUpdate_profileUpdate;
}

export interface profileUpdateVariables {
  input: ProfileUpdateInput;
}
