/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineOutputFields
// ====================================================

export interface deepRoutineOutputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineOutputFields_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineOutputFields_standard_tags_translations[];
}

export interface deepRoutineOutputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineOutputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineOutputFields_standard_tags[];
  translations: deepRoutineOutputFields_standard_translations[];
}

export interface deepRoutineOutputFields {
  __typename: "OutputItem";
  id: string;
  standard: deepRoutineOutputFields_standard | null;
}
