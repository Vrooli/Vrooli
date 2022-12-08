/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarRoutineFields
// ====================================================

export interface listStarRoutineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface listStarRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarRoutineFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarRoutineFields_tags_translations[];
}

export interface listStarRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface listStarRoutineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: listStarRoutineFields_permissionsRoutine;
  tags: listStarRoutineFields_tags[];
  translations: listStarRoutineFields_translations[];
}
