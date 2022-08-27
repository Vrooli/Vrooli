/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runInputDataFields
// ====================================================

export interface runInputDataFields_input_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runInputDataFields_input_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputDataFields_input_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runInputDataFields_input_standard_tags_translations[];
}

export interface runInputDataFields_input_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputDataFields_input_standard {
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
  tags: runInputDataFields_input_standard_tags[];
  translations: runInputDataFields_input_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runInputDataFields_input {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runInputDataFields_input_translations[];
  standard: runInputDataFields_input_standard | null;
}

export interface runInputDataFields {
  __typename: "RunInput";
  id: string;
  data: string;
  input: runInputDataFields_input;
}
