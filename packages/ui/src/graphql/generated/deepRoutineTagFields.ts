/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineTagFields
// ====================================================

export interface deepRoutineTagFields_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineTagFields {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineTagFields_translations[];
}
