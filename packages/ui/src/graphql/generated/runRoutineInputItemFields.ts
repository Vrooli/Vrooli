/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineInputItemFields
// ====================================================

export interface runRoutineInputItemFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineInputItemFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputItemFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineInputItemFields_standard_tags_translations[];
}

export interface runRoutineInputItemFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputItemFields_standard {
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
  tags: runRoutineInputItemFields_standard_tags[];
  translations: runRoutineInputItemFields_standard_translations[];
}

export interface runRoutineInputItemFields {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineInputItemFields_translations[];
  standard: runRoutineInputItemFields_standard | null;
}
