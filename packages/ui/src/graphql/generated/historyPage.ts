/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { HistoryPageInput, RunStatus, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: historyPage
// ====================================================

export interface historyPage_historyPage_activeRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_activeRuns_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_activeRuns_routine_tags_translations[];
}

export interface historyPage_historyPage_activeRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface historyPage_historyPage_activeRuns_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: historyPage_historyPage_activeRuns_routine_tags[];
  translations: historyPage_historyPage_activeRuns_routine_translations[];
  version: string | null;
}

export interface historyPage_historyPage_activeRuns {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: historyPage_historyPage_activeRuns_routine | null;
}

export interface historyPage_historyPage_completedRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_completedRuns_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_completedRuns_routine_tags_translations[];
}

export interface historyPage_historyPage_completedRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface historyPage_historyPage_completedRuns_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: historyPage_historyPage_completedRuns_routine_tags[];
  translations: historyPage_historyPage_completedRuns_routine_translations[];
  version: string | null;
}

export interface historyPage_historyPage_completedRuns {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: historyPage_historyPage_completedRuns_routine | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyViewed_to_Organization_tags_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: historyPage_historyPage_recentlyViewed_to_Organization_tags[];
  translations: historyPage_historyPage_recentlyViewed_to_Organization_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyViewed_to_Project_tags_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: historyPage_historyPage_recentlyViewed_to_Project_tags[];
  translations: historyPage_historyPage_recentlyViewed_to_Project_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyViewed_to_Routine_tags_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface historyPage_historyPage_recentlyViewed_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: historyPage_historyPage_recentlyViewed_to_Routine_tags[];
  translations: historyPage_historyPage_recentlyViewed_to_Routine_translations[];
  version: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyViewed_to_Standard_tags_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface historyPage_historyPage_recentlyViewed_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: historyPage_historyPage_recentlyViewed_to_Standard_tags[];
  translations: historyPage_historyPage_recentlyViewed_to_Standard_translations[];
}

export interface historyPage_historyPage_recentlyViewed_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type historyPage_historyPage_recentlyViewed_to = historyPage_historyPage_recentlyViewed_to_Organization | historyPage_historyPage_recentlyViewed_to_Project | historyPage_historyPage_recentlyViewed_to_Routine | historyPage_historyPage_recentlyViewed_to_Standard | historyPage_historyPage_recentlyViewed_to_User;

export interface historyPage_historyPage_recentlyViewed {
  __typename: "View";
  id: string;
  lastViewed: any;
  title: string;
  to: historyPage_historyPage_recentlyViewed_to;
}

export interface historyPage_historyPage_recentlyStarred_to_Tag {
  __typename: "Tag";
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Project_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Routine_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type historyPage_historyPage_recentlyStarred_to_Comment_commentedOn = historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Project | historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Routine | historyPage_historyPage_recentlyStarred_to_Comment_commentedOn_Standard;

export interface historyPage_historyPage_recentlyStarred_to_Comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: historyPage_historyPage_recentlyStarred_to_Comment_creator_Organization_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type historyPage_historyPage_recentlyStarred_to_Comment_creator = historyPage_historyPage_recentlyStarred_to_Comment_creator_Organization | historyPage_historyPage_recentlyStarred_to_Comment_creator_User;

export interface historyPage_historyPage_recentlyStarred_to_Comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface historyPage_historyPage_recentlyStarred_to_Comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  isStarred: boolean;
  commentedOn: historyPage_historyPage_recentlyStarred_to_Comment_commentedOn;
  creator: historyPage_historyPage_recentlyStarred_to_Comment_creator | null;
  translations: historyPage_historyPage_recentlyStarred_to_Comment_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyStarred_to_Organization_tags_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: historyPage_historyPage_recentlyStarred_to_Organization_tags[];
  translations: historyPage_historyPage_recentlyStarred_to_Organization_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyStarred_to_Project_tags_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: historyPage_historyPage_recentlyStarred_to_Project_tags[];
  translations: historyPage_historyPage_recentlyStarred_to_Project_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyStarred_to_Routine_tags_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface historyPage_historyPage_recentlyStarred_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: historyPage_historyPage_recentlyStarred_to_Routine_tags[];
  translations: historyPage_historyPage_recentlyStarred_to_Routine_translations[];
  version: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: historyPage_historyPage_recentlyStarred_to_Standard_tags_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface historyPage_historyPage_recentlyStarred_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: historyPage_historyPage_recentlyStarred_to_Standard_tags[];
  translations: historyPage_historyPage_recentlyStarred_to_Standard_translations[];
}

export interface historyPage_historyPage_recentlyStarred_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type historyPage_historyPage_recentlyStarred_to = historyPage_historyPage_recentlyStarred_to_Tag | historyPage_historyPage_recentlyStarred_to_Comment | historyPage_historyPage_recentlyStarred_to_Organization | historyPage_historyPage_recentlyStarred_to_Project | historyPage_historyPage_recentlyStarred_to_Routine | historyPage_historyPage_recentlyStarred_to_Standard | historyPage_historyPage_recentlyStarred_to_User;

export interface historyPage_historyPage_recentlyStarred {
  __typename: "Star";
  id: string;
  to: historyPage_historyPage_recentlyStarred_to;
}

export interface historyPage_historyPage {
  __typename: "HistoryPageResult";
  activeRuns: historyPage_historyPage_activeRuns[];
  completedRuns: historyPage_historyPage_completedRuns[];
  recentlyViewed: historyPage_historyPage_recentlyViewed[];
  recentlyStarred: historyPage_historyPage_recentlyStarred[];
}

export interface historyPage {
  historyPage: historyPage_historyPage;
}

export interface historyPageVariables {
  input: HistoryPageInput;
}
