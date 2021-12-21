/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { EmailLogInInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: emailLogIn
// ====================================================

export interface emailLogIn_emailLogIn_roles {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface emailLogIn_emailLogIn {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: emailLogIn_emailLogIn_roles[];
}

export interface emailLogIn {
  emailLogIn: emailLogIn_emailLogIn;
}

export interface emailLogInVariables {
  input?: EmailLogInInput | null;
}
