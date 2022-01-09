/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AutocompleteInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: autocomplete
// ====================================================

export interface autocomplete_autocomplete_organizations {
  __typename: "Organization";
  id: string;
  name: string;
  stars: number;
}

export interface autocomplete_autocomplete_projects {
  __typename: "Project";
  id: string;
  name: string;
  stars: number;
}

export interface autocomplete_autocomplete_routines {
  __typename: "Routine";
  id: string;
  title: string | null;
  stars: number;
}

export interface autocomplete_autocomplete_standards {
  __typename: "Standard";
  id: string;
  name: string;
  stars: number;
}

export interface autocomplete_autocomplete_users {
  __typename: "User";
  id: string;
  username: string | null;
  stars: number;
}

export interface autocomplete_autocomplete {
  __typename: "AutocompleteResult";
  organizations: autocomplete_autocomplete_organizations[];
  projects: autocomplete_autocomplete_projects[];
  routines: autocomplete_autocomplete_routines[];
  standards: autocomplete_autocomplete_standards[];
  users: autocomplete_autocomplete_users[];
}

export interface autocomplete {
  autocomplete: autocomplete_autocomplete;
}

export interface autocompleteVariables {
  input: AutocompleteInput;
}
