/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: nodeTagFields
// ====================================================

export interface nodeTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeTagFields {
  __typename: "Tag";
  tag: string;
  translations: nodeTagFields_translations[];
}
