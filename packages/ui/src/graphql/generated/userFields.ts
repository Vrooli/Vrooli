/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: userFields
// ====================================================

export interface userFields_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface userFields {
  __typename: "User";
  id: string;
  username: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
  translations: userFields_translations[];
}
