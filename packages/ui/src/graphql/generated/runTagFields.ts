/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runTagFields
// ====================================================

export interface runTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runTagFields {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runTagFields_translations[];
}
