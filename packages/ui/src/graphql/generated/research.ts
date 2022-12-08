/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: research
// ====================================================

export interface research_research_processes_permissionsRoutine {
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

export interface research_research_processes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_processes_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_processes_tags_translations[];
}

export interface research_research_processes_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface research_research_processes {
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
  permissionsRoutine: research_research_processes_permissionsRoutine;
  tags: research_research_processes_tags[];
  translations: research_research_processes_translations[];
}

export interface research_research_newlyCompleted_Project_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface research_research_newlyCompleted_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_newlyCompleted_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_newlyCompleted_Project_tags_translations[];
}

export interface research_research_newlyCompleted_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface research_research_newlyCompleted_Project {
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
  permissionsProject: research_research_newlyCompleted_Project_permissionsProject;
  tags: research_research_newlyCompleted_Project_tags[];
  translations: research_research_newlyCompleted_Project_translations[];
}

export interface research_research_newlyCompleted_Routine_permissionsRoutine {
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

export interface research_research_newlyCompleted_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_newlyCompleted_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_newlyCompleted_Routine_tags_translations[];
}

export interface research_research_newlyCompleted_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface research_research_newlyCompleted_Routine {
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
  permissionsRoutine: research_research_newlyCompleted_Routine_permissionsRoutine;
  tags: research_research_newlyCompleted_Routine_tags[];
  translations: research_research_newlyCompleted_Routine_translations[];
}

export type research_research_newlyCompleted = research_research_newlyCompleted_Project | research_research_newlyCompleted_Routine;

export interface research_research_needVotes_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface research_research_needVotes_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_needVotes_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_needVotes_tags_translations[];
}

export interface research_research_needVotes_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface research_research_needVotes {
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
  permissionsProject: research_research_needVotes_permissionsProject;
  tags: research_research_needVotes_tags[];
  translations: research_research_needVotes_translations[];
}

export interface research_research_needInvestments_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface research_research_needInvestments_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_needInvestments_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_needInvestments_tags_translations[];
}

export interface research_research_needInvestments_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface research_research_needInvestments {
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
  permissionsProject: research_research_needInvestments_permissionsProject;
  tags: research_research_needInvestments_tags[];
  translations: research_research_needInvestments_translations[];
}

export interface research_research_needMembers_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface research_research_needMembers_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface research_research_needMembers_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: research_research_needMembers_tags_translations[];
}

export interface research_research_needMembers_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface research_research_needMembers {
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
  permissionsOrganization: research_research_needMembers_permissionsOrganization | null;
  tags: research_research_needMembers_tags[];
  translations: research_research_needMembers_translations[];
}

export interface research_research {
  __typename: "ResearchResult";
  processes: research_research_processes[];
  newlyCompleted: research_research_newlyCompleted[];
  needVotes: research_research_needVotes[];
  needInvestments: research_research_needInvestments[];
  needMembers: research_research_needMembers[];
}

export interface research {
  research: research_research;
}
