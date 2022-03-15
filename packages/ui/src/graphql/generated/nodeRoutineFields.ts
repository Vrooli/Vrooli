/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeRoutineFields
// ====================================================

export interface nodeRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeRoutineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: nodeRoutineFields_tags_translations[];
}

export interface nodeRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface nodeRoutineFields {
  __typename: "Routine";
  id: string;
  complexity: number;
  version: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  role: MemberRole | null;
  simplicity: number;
  tags: nodeRoutineFields_tags[];
  translations: nodeRoutineFields_translations[];
}
