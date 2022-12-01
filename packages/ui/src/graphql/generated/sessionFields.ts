/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: sessionFields
// ====================================================

export interface sessionFields_users {
  __typename: "SessionUser";
  handle: string | null;
  id: string;
  languages: string[];
  name: string | null;
  theme: string | null;
}

export interface sessionFields {
  __typename: "Session";
  isLoggedIn: boolean;
  users: sessionFields_users[] | null;
}
