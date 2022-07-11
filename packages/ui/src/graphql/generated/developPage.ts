/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: developPage
// ====================================================

export interface developPage_developPage_completed_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Project_tags {
  __typename: "Tag";
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
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  tags: developPage_developPage_completed_Project_tags[];
  translations: developPage_developPage_completed_Project_translations[];
}

export interface developPage_developPage_completed_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_completed_Routine_tags {
  __typename: "Tag";
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
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: developPage_developPage_completed_Routine_tags[];
  translations: developPage_developPage_completed_Routine_translations[];
  version: string | null;
}

export type developPage_developPage_completed = developPage_developPage_completed_Project | developPage_developPage_completed_Routine;

export interface developPage_developPage_inProgress_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Project_tags {
  __typename: "Tag";
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
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  tags: developPage_developPage_inProgress_Project_tags[];
  translations: developPage_developPage_inProgress_Project_translations[];
}

export interface developPage_developPage_inProgress_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_inProgress_Routine_tags {
  __typename: "Tag";
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
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: developPage_developPage_inProgress_Routine_tags[];
  translations: developPage_developPage_inProgress_Routine_translations[];
  version: string | null;
}

export type developPage_developPage_inProgress = developPage_developPage_inProgress_Project | developPage_developPage_inProgress_Routine;

export interface developPage_developPage_recent_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Project_tags {
  __typename: "Tag";
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
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  tags: developPage_developPage_recent_Project_tags[];
  translations: developPage_developPage_recent_Project_translations[];
}

export interface developPage_developPage_recent_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface developPage_developPage_recent_Routine_tags {
  __typename: "Tag";
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
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  reportsCount: number;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: developPage_developPage_recent_Routine_tags[];
  translations: developPage_developPage_recent_Routine_translations[];
  version: string | null;
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
