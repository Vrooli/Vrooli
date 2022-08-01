/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: developPage
// ====================================================

export interface developPage_developPage_completed_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface developPage_developPage_completed_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_completed_Project_tags_translations[];
}

export interface developPage_developPage_completed_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface developPage_developPage_completed_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: developPage_developPage_completed_Project_permissionsProject;
  tags: developPage_developPage_completed_Project_tags[];
  translations: developPage_developPage_completed_Project_translations[];
}

export interface developPage_developPage_completed_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface developPage_developPage_completed_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_completed_Routine_tags_translations[];
}

export interface developPage_developPage_completed_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_completed_Routine {
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
  permissionsRoutine: developPage_developPage_completed_Routine_permissionsRoutine;
  tags: developPage_developPage_completed_Routine_tags[];
  translations: developPage_developPage_completed_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export type developPage_developPage_completed = developPage_developPage_completed_Project | developPage_developPage_completed_Routine;

export interface developPage_developPage_inProgress_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface developPage_developPage_inProgress_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_inProgress_Project_tags_translations[];
}

export interface developPage_developPage_inProgress_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: developPage_developPage_inProgress_Project_permissionsProject;
  tags: developPage_developPage_inProgress_Project_tags[];
  translations: developPage_developPage_inProgress_Project_translations[];
}

export interface developPage_developPage_inProgress_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface developPage_developPage_inProgress_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_inProgress_Routine_tags_translations[];
}

export interface developPage_developPage_inProgress_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_inProgress_Routine {
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
  permissionsRoutine: developPage_developPage_inProgress_Routine_permissionsRoutine;
  tags: developPage_developPage_inProgress_Routine_tags[];
  translations: developPage_developPage_inProgress_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export type developPage_developPage_inProgress = developPage_developPage_inProgress_Project | developPage_developPage_inProgress_Routine;

export interface developPage_developPage_recent_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface developPage_developPage_recent_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_recent_Project_tags_translations[];
}

export interface developPage_developPage_recent_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface developPage_developPage_recent_Project {
  __typename: "Project";
  id: string;
  commentsCount: number;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: developPage_developPage_recent_Project_permissionsProject;
  tags: developPage_developPage_recent_Project_tags[];
  translations: developPage_developPage_recent_Project_translations[];
}

export interface developPage_developPage_recent_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface developPage_developPage_recent_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: developPage_developPage_recent_Routine_tags_translations[];
}

export interface developPage_developPage_recent_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface developPage_developPage_recent_Routine {
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
  permissionsRoutine: developPage_developPage_recent_Routine_permissionsRoutine;
  tags: developPage_developPage_recent_Routine_tags[];
  translations: developPage_developPage_recent_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export type developPage_developPage_recent = developPage_developPage_recent_Project | developPage_developPage_recent_Routine;

export interface developPage_developPage {
  __typename: "DevelopPageResult";
  completed: developPage_developPage_completed[];
  inProgress: developPage_developPage_inProgress[];
  recent: developPage_developPage_recent[];
}

export interface developPage {
  developPage: developPage_developPage;
}
