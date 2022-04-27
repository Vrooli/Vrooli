/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: researchPage
// ====================================================

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
  tags: researchPage_researchPage_processes_tags[];
  translations: researchPage_researchPage_processes_translations[];
  version: string | null;
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
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: researchPage_researchPage_newlyCompleted_Project_tags[];
  translations: researchPage_researchPage_newlyCompleted_Project_translations[];
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
  tags: researchPage_researchPage_newlyCompleted_Routine_tags[];
  translations: researchPage_researchPage_newlyCompleted_Routine_translations[];
  version: string | null;
}

export type researchPage_researchPage_newlyCompleted = researchPage_researchPage_newlyCompleted_Project | researchPage_researchPage_newlyCompleted_Routine;

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
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: researchPage_researchPage_needVotes_tags[];
  translations: researchPage_researchPage_needVotes_translations[];
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
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: researchPage_researchPage_needInvestments_tags[];
  translations: researchPage_researchPage_needInvestments_translations[];
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
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
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
