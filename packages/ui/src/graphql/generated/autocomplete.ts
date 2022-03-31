/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AutocompleteInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: autocomplete
// ====================================================

export interface autocomplete_autocomplete_organizations_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface autocomplete_autocomplete_organizations_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: autocomplete_autocomplete_organizations_tags_translations[];
}

export interface autocomplete_autocomplete_organizations_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface autocomplete_autocomplete_organizations {
  __typename: "Organization";
  id: string;
  stars: number;
  isStarred: boolean;
  tags: autocomplete_autocomplete_organizations_tags[];
  translations: autocomplete_autocomplete_organizations_translations[];
}

export interface autocomplete_autocomplete_projects_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface autocomplete_autocomplete_projects_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: autocomplete_autocomplete_projects_tags_translations[];
}

export interface autocomplete_autocomplete_projects_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface autocomplete_autocomplete_projects {
  __typename: "Project";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: autocomplete_autocomplete_projects_tags[];
  translations: autocomplete_autocomplete_projects_translations[];
}

export interface autocomplete_autocomplete_routines_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface autocomplete_autocomplete_routines_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: autocomplete_autocomplete_routines_tags_translations[];
}

export interface autocomplete_autocomplete_routines_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface autocomplete_autocomplete_routines {
  __typename: "Routine";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: autocomplete_autocomplete_routines_tags[];
  translations: autocomplete_autocomplete_routines_translations[];
}

export interface autocomplete_autocomplete_standards_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface autocomplete_autocomplete_standards_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: autocomplete_autocomplete_standards_tags_translations[];
}

export interface autocomplete_autocomplete_standards_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface autocomplete_autocomplete_standards {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  tags: autocomplete_autocomplete_standards_tags[];
  translations: autocomplete_autocomplete_standards_translations[];
}

export interface autocomplete_autocomplete_users {
  __typename: "User";
  id: string;
  username: string | null;
  stars: number;
  isStarred: boolean;
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
