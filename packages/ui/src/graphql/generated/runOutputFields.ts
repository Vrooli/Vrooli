/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: runOutputFields
// ====================================================

export interface runOutputFields_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runOutputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runOutputFields_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runOutputFields_standard_tags_translations[];
}

export interface runOutputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runOutputFields_standard {
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
  tags: runOutputFields_standard_tags[];
  translations: runOutputFields_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface runOutputFields {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runOutputFields_translations[];
  standard: runOutputFields_standard | null;
}
