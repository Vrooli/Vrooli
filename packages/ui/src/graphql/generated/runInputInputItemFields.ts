/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runInputInputItemFields
// ====================================================

export interface runInputInputItemFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runInputInputItemFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputInputItemFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runInputInputItemFields_standard_tags_translations[];
}

export interface runInputInputItemFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputInputItemFields_standard {
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
  tags: runInputInputItemFields_standard_tags[];
  translations: runInputInputItemFields_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runInputInputItemFields {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runInputInputItemFields_translations[];
  standard: runInputInputItemFields_standard | null;
}
