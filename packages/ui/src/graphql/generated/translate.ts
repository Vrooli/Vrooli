/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TranslateInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: translate
// ====================================================

export interface translate_translate {
  __typename: "Translate";
  fields: string;
  language: string;
}

export interface translate {
  translate: translate_translate;
}

export interface translateVariables {
  input: TranslateInput;
}
