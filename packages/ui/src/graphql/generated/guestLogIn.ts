/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL mutation operation: guestLogIn
// ====================================================

export interface guestLogIn_guestLogIn_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: (string | null)[] | null;
  name: string | null;
  theme: string | null;
}

export interface guestLogIn_guestLogIn {
  __typename: "Session";
  isLoggedIn: boolean;
  users: guestLogIn_guestLogIn_users[] | null;
}

export interface guestLogIn {
  guestLogIn: guestLogIn_guestLogIn;
}
