/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineResourceFields
// ====================================================

export interface deepRoutineResourceFields_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface deepRoutineResourceFields {
  __typename: "Resource";
  id: string;
  created_at: any;
  link: string;
  updated_at: any;
  translations: deepRoutineResourceFields_translations[];
}
