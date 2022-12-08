/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listRunRoutineRoutineFields
// ====================================================

export interface listRunRoutineRoutineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface listRunRoutineRoutineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunRoutineRoutineFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listRunRoutineRoutineFields_tags_translations[];
}

export interface listRunRoutineRoutineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface listRunRoutineRoutineFields {
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
  permissionsRoutine: listRunRoutineRoutineFields_permissionsRoutine;
  tags: listRunRoutineRoutineFields_tags[];
  translations: listRunRoutineRoutineFields_translations[];
}
