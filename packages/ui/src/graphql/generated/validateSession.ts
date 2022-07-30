/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL mutation operation: validateSession
// ====================================================

export interface validateSession_validateSession {
  __typename: "Session";
  id: string | null;
  theme: string;
  isLoggedIn: boolean;
  languages: string[] | null;
}

export interface validateSession {
  validateSession: validateSession_validateSession;
}
