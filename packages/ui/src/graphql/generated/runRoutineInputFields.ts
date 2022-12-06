/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runRoutineInputFields
// ====================================================

export interface runRoutineInputFields_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineInputFields_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputFields_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineInputFields_input_standard_tags_translations[];
}

export interface runRoutineInputFields_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineInputFields_input_standard {
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
  tags: runRoutineInputFields_input_standard_tags[];
  translations: runRoutineInputFields_input_standard_translations[];
}

export interface runRoutineInputFields_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineInputFields_input_translations[];
  standard: runRoutineInputFields_input_standard | null;
}

export interface runRoutineInputFields {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: runRoutineInputFields_input;
}
