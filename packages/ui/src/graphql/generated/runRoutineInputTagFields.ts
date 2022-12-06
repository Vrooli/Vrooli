/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineInputTagFields
// ====================================================

export interface runRoutineInputTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputTagFields {
  __typename: "Tag";
  tag: string;
  translations: runRoutineInputTagFields_translations[];
}
