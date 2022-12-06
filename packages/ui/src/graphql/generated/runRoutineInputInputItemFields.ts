/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineInputInputItemFields
// ====================================================

export interface runRoutineInputInputItemFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineInputInputItemFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputInputItemFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineInputInputItemFields_standard_tags_translations[];
}

export interface runRoutineInputInputItemFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputInputItemFields_standard {
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
  tags: runRoutineInputInputItemFields_standard_tags[];
  translations: runRoutineInputInputItemFields_standard_translations[];
}

export interface runRoutineInputInputItemFields {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineInputInputItemFields_translations[];
  standard: runRoutineInputInputItemFields_standard | null;
}
