/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: deepRoutineInputFields
// ====================================================

export interface deepRoutineInputFields_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineInputFields_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineInputFields_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: deepRoutineInputFields_standard_tags_translations[];
}

export interface deepRoutineInputFields_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface deepRoutineInputFields_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineInputFields_standard_tags[];
  translations: deepRoutineInputFields_standard_translations[];
}

export interface deepRoutineInputFields {
  __typename: "InputItem";
  id: string;
  name: string | null;
  translations: deepRoutineInputFields_translations[];
  standard: deepRoutineInputFields_standard | null;
}
