/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineTagFields
// ====================================================

export interface runRoutineTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineTagFields {
  __typename: "Tag";
  tag: string;
  translations: runRoutineTagFields_translations[];
}
