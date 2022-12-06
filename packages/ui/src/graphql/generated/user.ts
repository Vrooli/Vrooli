/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdOrHandleInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: user
// ====================================================

export interface user_user_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface user_user {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  created_at: any;
  stars: number;
  isStarred: boolean;
  reportsCount: number;
  translations: user_user_translations[];
}

export interface user {
  user: user_user | null;
}

export interface userVariables {
  input: FindByIdOrHandleInput;
}
