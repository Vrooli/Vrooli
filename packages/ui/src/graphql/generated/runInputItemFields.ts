/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runInputItemFields
// ====================================================

export interface runInputItemFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputItemFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputItemFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runInputItemFields_standard_tags_translations[];
}

export interface runInputItemFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runInputItemFields_standard {
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
  tags: runInputItemFields_standard_tags[];
  translations: runInputItemFields_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runInputItemFields {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runInputItemFields_translations[];
  standard: runInputItemFields_standard | null;
}
