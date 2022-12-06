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
  handle: string | null;
  name: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  reportsCount: number;
  translations: userFields_translations[];
}
