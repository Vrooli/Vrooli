/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL mutation operation: guestLogIn
// ====================================================

export interface guestLogIn_guestLogIn {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: string[];
  languages: string[] | null;
}

export interface guestLogIn {
  guestLogIn: guestLogIn_guestLogIn;
}
