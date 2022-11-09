/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ValidateSessionInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: validateSession
// ====================================================

export interface validateSession_validateSession_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: (string | null)[] | null;
  name: string | null;
  theme: string | null;
}

export interface validateSession_validateSession {
  __typename: "Session";
  isLoggedIn: boolean;
  users: validateSession_validateSession_users[] | null;
}

export interface validateSession {
  validateSession: validateSession_validateSession;
}

export interface validateSessionVariables {
  input: ValidateSessionInput;
}
