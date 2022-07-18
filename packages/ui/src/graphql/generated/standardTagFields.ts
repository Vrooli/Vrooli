/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: standardTagFields
// ====================================================

export interface standardTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardTagFields {
  __typename: "Tag";
  tag: string;
  translations: standardTagFields_translations[];
}
