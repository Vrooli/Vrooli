/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL mutation operation: validateSession
// ====================================================

export interface validateSession_validateSession_roles {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface validateSession_validateSession {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: validateSession_validateSession_roles[];
}

export interface validateSession {
  validateSession: validateSession_validateSession;
}
