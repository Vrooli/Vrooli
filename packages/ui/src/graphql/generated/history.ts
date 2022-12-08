/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { HistoryInput, RunStatus } from "./globalTypes";

// ====================================================
// GraphQL query operation: history
// ====================================================

export interface history_history_activeRuns_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface history_history_activeRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_activeRuns_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_activeRuns_routine_tags_translations[];
}

export interface history_history_activeRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name string;
}

export interface history_history_activeRuns_routine {
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
  permissionsRoutine: history_history_activeRuns_routine_permissionsRoutine;
  tags: history_history_activeRuns_routine_tags[];
  translations: history_history_activeRuns_routine_translations[];
}

export interface history_history_activeRuns {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  name string;
  status: RunStatus;
  routine: history_history_activeRuns_routine | null;
}

export interface history_history_completedRuns_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface history_history_completedRuns_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_completedRuns_routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_completedRuns_routine_tags_translations[];
}

export interface history_history_completedRuns_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name string;
}

export interface history_history_completedRuns_routine {
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
  permissionsRoutine: history_history_completedRuns_routine_permissionsRoutine;
  tags: history_history_completedRuns_routine_tags[];
  translations: history_history_completedRuns_routine_translations[];
}

export interface history_history_completedRuns {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  isPrivate: boolean;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  name string;
  status: RunStatus;
  routine: history_history_completedRuns_routine | null;
}

export interface history_history_recentlyViewed_to_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface history_history_recentlyViewed_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyViewed_to_Organization_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyViewed_to_Organization_tags_translations[];
}

export interface history_history_recentlyViewed_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface history_history_recentlyViewed_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: history_history_recentlyViewed_to_Organization_permissionsOrganization | null;
  tags: history_history_recentlyViewed_to_Organization_tags[];
  translations: history_history_recentlyViewed_to_Organization_translations[];
}

export interface history_history_recentlyViewed_to_Project_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface history_history_recentlyViewed_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyViewed_to_Project_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyViewed_to_Project_tags_translations[];
}

export interface history_history_recentlyViewed_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface history_history_recentlyViewed_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  permissionsProject: history_history_recentlyViewed_to_Project_permissionsProject;
  tags: history_history_recentlyViewed_to_Project_tags[];
  translations: history_history_recentlyViewed_to_Project_translations[];
}

export interface history_history_recentlyViewed_to_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface history_history_recentlyViewed_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyViewed_to_Routine_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyViewed_to_Routine_tags_translations[];
}

export interface history_history_recentlyViewed_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name string;
}

export interface history_history_recentlyViewed_to_Routine {
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
  permissionsRoutine: history_history_recentlyViewed_to_Routine_permissionsRoutine;
  tags: history_history_recentlyViewed_to_Routine_tags[];
  translations: history_history_recentlyViewed_to_Routine_translations[];
}

export interface history_history_recentlyViewed_to_Standard_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface history_history_recentlyViewed_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyViewed_to_Standard_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyViewed_to_Standard_tags_translations[];
}

export interface history_history_recentlyViewed_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface history_history_recentlyViewed_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: history_history_recentlyViewed_to_Standard_permissionsStandard;
  tags: history_history_recentlyViewed_to_Standard_tags[];
  translations: history_history_recentlyViewed_to_Standard_translations[];
}

export interface history_history_recentlyViewed_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type history_history_recentlyViewed_to = history_history_recentlyViewed_to_Organization | history_history_recentlyViewed_to_Project | history_history_recentlyViewed_to_Routine | history_history_recentlyViewed_to_Standard | history_history_recentlyViewed_to_User;

export interface history_history_recentlyViewed {
  __typename: "View";
  id: string;
  lastViewed: any;
  name string;
  to: history_history_recentlyViewed_to;
}

export interface history_history_recentlyStarred_to_Api {
  __typename: "Api" | "Issue" | "Note" | "Post" | "Question" | "QuestionAnswer" | "Quiz" | "SmartContract";
}

export interface history_history_recentlyStarred_to_Comment_commentedOn_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
}

export interface history_history_recentlyStarred_to_Comment_commentedOn_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  translations: history_history_recentlyStarred_to_Comment_commentedOn_Project_translations[];
}

export interface history_history_recentlyStarred_to_Comment_commentedOn_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name string;
}

export interface history_history_recentlyStarred_to_Comment_commentedOn_Routine {
  __typename: "Routine";
  id: string;
  translations: history_history_recentlyStarred_to_Comment_commentedOn_Routine_translations[];
}

export interface history_history_recentlyStarred_to_Comment_commentedOn_Standard {
  __typename: "Standard";
  id: string;
  name: string;
}

export type history_history_recentlyStarred_to_Comment_commentedOn = history_history_recentlyStarred_to_Comment_commentedOn_Project | history_history_recentlyStarred_to_Comment_commentedOn_Routine | history_history_recentlyStarred_to_Comment_commentedOn_Standard;

export interface history_history_recentlyStarred_to_Comment_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface history_history_recentlyStarred_to_Comment_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: history_history_recentlyStarred_to_Comment_creator_Organization_translations[];
}

export interface history_history_recentlyStarred_to_Comment_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type history_history_recentlyStarred_to_Comment_creator = history_history_recentlyStarred_to_Comment_creator_Organization | history_history_recentlyStarred_to_Comment_creator_User;

export interface history_history_recentlyStarred_to_Comment_permissionsComment {
  __typename: "CommentPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReply: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface history_history_recentlyStarred_to_Comment_translations {
  __typename: "CommentTranslation";
  id: string;
  language: string;
  text: string;
}

export interface history_history_recentlyStarred_to_Comment {
  __typename: "Comment";
  id: string;
  created_at: any;
  updated_at: any;
  score: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  commentedOn: history_history_recentlyStarred_to_Comment_commentedOn;
  creator: history_history_recentlyStarred_to_Comment_creator | null;
  permissionsComment: history_history_recentlyStarred_to_Comment_permissionsComment | null;
  translations: history_history_recentlyStarred_to_Comment_translations[];
}

export interface history_history_recentlyStarred_to_Organization_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface history_history_recentlyStarred_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyStarred_to_Organization_tags_translations[];
}

export interface history_history_recentlyStarred_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface history_history_recentlyStarred_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: history_history_recentlyStarred_to_Organization_permissionsOrganization | null;
  tags: history_history_recentlyStarred_to_Organization_tags[];
  translations: history_history_recentlyStarred_to_Organization_translations[];
}

export interface history_history_recentlyStarred_to_Project_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface history_history_recentlyStarred_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyStarred_to_Project_tags_translations[];
}

export interface history_history_recentlyStarred_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  permissionsProject: history_history_recentlyStarred_to_Project_permissionsProject;
  tags: history_history_recentlyStarred_to_Project_tags[];
  translations: history_history_recentlyStarred_to_Project_translations[];
}

export interface history_history_recentlyStarred_to_Routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface history_history_recentlyStarred_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyStarred_to_Routine_tags_translations[];
}

export interface history_history_recentlyStarred_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name string;
}

export interface history_history_recentlyStarred_to_Routine {
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
  permissionsRoutine: history_history_recentlyStarred_to_Routine_permissionsRoutine;
  tags: history_history_recentlyStarred_to_Routine_tags[];
  translations: history_history_recentlyStarred_to_Routine_translations[];
}

export interface history_history_recentlyStarred_to_Standard_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface history_history_recentlyStarred_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyStarred_to_Standard_tags_translations[];
}

export interface history_history_recentlyStarred_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: history_history_recentlyStarred_to_Standard_permissionsStandard;
  tags: history_history_recentlyStarred_to_Standard_tags[];
  translations: history_history_recentlyStarred_to_Standard_translations[];
}

export interface history_history_recentlyStarred_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export interface history_history_recentlyStarred_to_Tag_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface history_history_recentlyStarred_to_Tag {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: history_history_recentlyStarred_to_Tag_translations[];
}

export type history_history_recentlyStarred_to = history_history_recentlyStarred_to_Api | history_history_recentlyStarred_to_Comment | history_history_recentlyStarred_to_Organization | history_history_recentlyStarred_to_Project | history_history_recentlyStarred_to_Routine | history_history_recentlyStarred_to_Standard | history_history_recentlyStarred_to_User | history_history_recentlyStarred_to_Tag;

export interface history_history_recentlyStarred {
  __typename: "Star";
  id: string;
  to: history_history_recentlyStarred_to;
}

export interface history_history {
  __typename: "HistoryResult";
  activeRuns: history_history_activeRuns[];
  completedRuns: history_history_completedRuns[];
  recentlyViewed: history_history_recentlyViewed[];
  recentlyStarred: history_history_recentlyStarred[];
}

export interface history {
  history: history_history;
}

export interface historyVariables {
  input: HistoryInput;
}
