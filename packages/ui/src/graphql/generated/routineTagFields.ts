/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: routineTagFields
// ====================================================

export interface routineTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineTagFields {
  __typename: "Tag";
  tag: string;
  translations: routineTagFields_translations[];
}
