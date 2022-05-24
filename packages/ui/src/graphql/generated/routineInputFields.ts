/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: routineInputFields
// ====================================================

export interface routineInputFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineInputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineInputFields_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineInputFields_standard_tags_translations[];
}

export interface routineInputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineInputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: routineInputFields_standard_tags[];
  translations: routineInputFields_standard_translations[];
  version: string;
}

export interface routineInputFields {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineInputFields_translations[];
  standard: routineInputFields_standard | null;
}
