/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: projectTagFields
// ====================================================

export interface projectTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface projectTagFields {
  __typename: "Tag";
  tag: string;
  translations: projectTagFields_translations[];
}
