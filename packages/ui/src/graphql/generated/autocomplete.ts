/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AutocompleteInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: autocomplete
// ====================================================

export interface autocomplete_autocomplete {
  __typename: "Autocomplete";
  id: string;
  title: string;
  objectType: string;
  stars: number;
}

export interface autocomplete {
  autocomplete: autocomplete_autocomplete[];
}

export interface autocompleteVariables {
  input: AutocompleteInput;
}
