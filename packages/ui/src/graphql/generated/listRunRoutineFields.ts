/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: listRunRoutineFields
// ====================================================

export interface listRunRoutineFields_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface listRunRoutineFields_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listRunRoutineFields_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listRunRoutineFields_routine_tags_translations[];
}

export interface listRunRoutineFields_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface listRunRoutineFields_routine {
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
  permissionsRoutine: listRunRoutineFields_routine_permissionsRoutine;
  tags: listRunRoutineFields_routine_tags[];
  translations: listRunRoutineFields_routine_translations[];
}

export interface listRunRoutineFields {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunStatus;
  routine: listRunRoutineFields_routine | null;
}
