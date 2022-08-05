/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: nodeRoutineFields
// ====================================================

export interface nodeRoutineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface nodeRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeRoutineFields_tags {
  __typename: "Tag";
  tag: string;
  translations: nodeRoutineFields_tags_translations[];
}

export interface nodeRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface nodeRoutineFields {
  __typename: "Routine";
  id: string;
  complexity: number;
  version: string;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: nodeRoutineFields_permissionsRoutine;
  tags: nodeRoutineFields_tags[];
  translations: nodeRoutineFields_translations[];
}
