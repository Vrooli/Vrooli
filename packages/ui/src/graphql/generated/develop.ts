/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: develop
// ====================================================

export interface develop_develop_completed_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface develop_develop_completed_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_completed_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_completed_Project_tags_translations[];
}

export interface develop_develop_completed_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface develop_develop_completed_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: develop_develop_completed_Project_permissionsProject;
  tags: develop_develop_completed_Project_tags[];
  translations: develop_develop_completed_Project_translations[];
}

export interface develop_develop_completed_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface develop_develop_completed_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_completed_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_completed_Routine_tags_translations[];
}

export interface develop_develop_completed_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface develop_develop_completed_Routine {
  __typename: "Routine";
  id: string;
  commentsCount: number;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: develop_develop_completed_Routine_permissionsRoutine;
  tags: develop_develop_completed_Routine_tags[];
  translations: develop_develop_completed_Routine_translations[];
}

export type develop_develop_completed = develop_develop_completed_Project | develop_develop_completed_Routine;

export interface develop_develop_inProgress_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface develop_develop_inProgress_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_inProgress_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_inProgress_Project_tags_translations[];
}

export interface develop_develop_inProgress_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface develop_develop_inProgress_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: develop_develop_inProgress_Project_permissionsProject;
  tags: develop_develop_inProgress_Project_tags[];
  translations: develop_develop_inProgress_Project_translations[];
}

export interface develop_develop_inProgress_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface develop_develop_inProgress_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_inProgress_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_inProgress_Routine_tags_translations[];
}

export interface develop_develop_inProgress_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface develop_develop_inProgress_Routine {
  __typename: "Routine";
  id: string;
  commentsCount: number;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: develop_develop_inProgress_Routine_permissionsRoutine;
  tags: develop_develop_inProgress_Routine_tags[];
  translations: develop_develop_inProgress_Routine_translations[];
}

export type develop_develop_inProgress = develop_develop_inProgress_Project | develop_develop_inProgress_Routine;

export interface develop_develop_recent_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface develop_develop_recent_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_recent_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_recent_Project_tags_translations[];
}

export interface develop_develop_recent_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface develop_develop_recent_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: develop_develop_recent_Project_permissionsProject;
  tags: develop_develop_recent_Project_tags[];
  translations: develop_develop_recent_Project_translations[];
}

export interface develop_develop_recent_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface develop_develop_recent_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface develop_develop_recent_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: develop_develop_recent_Routine_tags_translations[];
}

export interface develop_develop_recent_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface develop_develop_recent_Routine {
  __typename: "Routine";
  id: string;
  commentsCount: number;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isDeleted: boolean;
  isInternal: boolean | null;
  isComplete: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: develop_develop_recent_Routine_permissionsRoutine;
  tags: develop_develop_recent_Routine_tags[];
  translations: develop_develop_recent_Routine_translations[];
}

export type develop_develop_recent = develop_develop_recent_Project | develop_develop_recent_Routine;

export interface develop_develop {
  __typename: "DevelopResult";
  completed: develop_develop_completed[];
  inProgress: develop_develop_inProgress[];
  recent: develop_develop_recent[];
}

export interface develop {
  develop: develop_develop;
}
