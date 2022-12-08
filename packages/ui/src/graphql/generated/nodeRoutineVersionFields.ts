/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: nodeRoutineVersionFields
// ====================================================

export interface nodeRoutineVersionFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface nodeRoutineVersionFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeRoutineVersionFields_tags {
  __typename: "Tag";
  tag: string;
  translations: nodeRoutineVersionFields_tags_translations[];
}

export interface nodeRoutineVersionFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
}

export interface nodeRoutineVersionFields {
  __typename: "Routine";
  id: string;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: nodeRoutineVersionFields_permissionsRoutine;
  tags: nodeRoutineVersionFields_tags[];
  translations: nodeRoutineVersionFields_translations[];
}
