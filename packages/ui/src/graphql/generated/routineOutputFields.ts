/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: routineOutputFields
// ====================================================

export interface routineOutputFields_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineOutputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineOutputFields_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineOutputFields_standard_tags_translations[];
}

export interface routineOutputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineOutputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: routineOutputFields_standard_tags[];
  translations: routineOutputFields_standard_translations[];
}

export interface routineOutputFields {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineOutputFields_translations[];
  standard: routineOutputFields_standard | null;
}
