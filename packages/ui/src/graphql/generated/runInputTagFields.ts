/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runInputTagFields
// ====================================================

export interface runInputTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputTagFields {
  __typename: "Tag";
  tag: string;
  translations: runInputTagFields_translations[];
}
