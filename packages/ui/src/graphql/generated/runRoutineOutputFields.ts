/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineOutputFields
// ====================================================

export interface runRoutineOutputFields_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineOutputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineOutputFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineOutputFields_standard_tags_translations[];
}

export interface runRoutineOutputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineOutputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runRoutineOutputFields_standard_tags[];
  translations: runRoutineOutputFields_standard_translations[];
}

export interface runRoutineOutputFields {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineOutputFields_translations[];
  standard: runRoutineOutputFields_standard | null;
}
