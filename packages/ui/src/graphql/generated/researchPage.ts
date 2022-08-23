/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: researchPage
// ====================================================

export interface researchPage_researchPage_processes_permissionsRoutine {
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

export interface researchPage_researchPage_processes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_processes_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_processes_tags_translations[];
}

export interface researchPage_researchPage_processes_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_processes {
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
  permissionsRoutine: researchPage_researchPage_processes_permissionsRoutine;
  tags: researchPage_researchPage_processes_tags[];
  translations: researchPage_researchPage_processes_translations[];
  version: string;
  versionGroupId: string;
}

export interface researchPage_researchPage_newlyCompleted_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface researchPage_researchPage_newlyCompleted_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Project_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Project {
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
  permissionsProject: researchPage_researchPage_newlyCompleted_Project_permissionsProject;
  tags: researchPage_researchPage_newlyCompleted_Project_tags[];
  translations: researchPage_researchPage_newlyCompleted_Project_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_permissionsRoutine {
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

export interface researchPage_researchPage_newlyCompleted_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_newlyCompleted_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_newlyCompleted_Routine_tags_translations[];
}

export interface researchPage_researchPage_newlyCompleted_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface researchPage_researchPage_newlyCompleted_Routine {
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
  permissionsRoutine: researchPage_researchPage_newlyCompleted_Routine_permissionsRoutine;
  tags: researchPage_researchPage_newlyCompleted_Routine_tags[];
  translations: researchPage_researchPage_newlyCompleted_Routine_translations[];
  version: string;
  versionGroupId: string;
}

export type researchPage_researchPage_newlyCompleted = researchPage_researchPage_newlyCompleted_Project | researchPage_researchPage_newlyCompleted_Routine;

export interface researchPage_researchPage_needVotes_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface researchPage_researchPage_needVotes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_needVotes_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_needVotes_tags_translations[];
}

export interface researchPage_researchPage_needVotes_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface researchPage_researchPage_needVotes {
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
  permissionsProject: researchPage_researchPage_needVotes_permissionsProject;
  tags: researchPage_researchPage_needVotes_tags[];
  translations: researchPage_researchPage_needVotes_translations[];
}

export interface researchPage_researchPage_needInvestments_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface researchPage_researchPage_needInvestments_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_needInvestments_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_needInvestments_tags_translations[];
}

export interface researchPage_researchPage_needInvestments_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface researchPage_researchPage_needInvestments {
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
  permissionsProject: researchPage_researchPage_needInvestments_permissionsProject;
  tags: researchPage_researchPage_needInvestments_tags[];
  translations: researchPage_researchPage_needInvestments_translations[];
}

export interface researchPage_researchPage_needMembers_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface researchPage_researchPage_needMembers_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface researchPage_researchPage_needMembers_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: researchPage_researchPage_needMembers_tags_translations[];
}

export interface researchPage_researchPage_needMembers_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface researchPage_researchPage_needMembers {
  __typename: "Organization";
  id: string;
  commentsCount: number;
  handle: string | null;
  stars: number;
  isOpenToNewMembers: boolean;
  isPrivate: boolean;
  isStarred: boolean;
  membersCount: number;
  reportsCount: number;
  permissionsOrganization: researchPage_researchPage_needMembers_permissionsOrganization | null;
  tags: researchPage_researchPage_needMembers_tags[];
  translations: researchPage_researchPage_needMembers_translations[];
}

export interface researchPage_researchPage {
  __typename: "ResearchPageResult";
  processes: researchPage_researchPage_processes[];
  newlyCompleted: researchPage_researchPage_newlyCompleted[];
  needVotes: researchPage_researchPage_needVotes[];
  needInvestments: researchPage_researchPage_needInvestments[];
  needMembers: researchPage_researchPage_needMembers[];
}

export interface researchPage {
  researchPage: researchPage_researchPage;
}
