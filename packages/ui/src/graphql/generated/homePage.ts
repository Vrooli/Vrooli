/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { HomePageInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: homePage
// ====================================================

export interface homePage_homePage_organizations_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface homePage_homePage_organizations_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: homePage_homePage_organizations_tags_translations[];
}

export interface homePage_homePage_organizations_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface homePage_homePage_organizations {
  __typename: "Organization";
  id: string;
  commentsCount: number;
  handle: string | null;
  stars: number;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  reportsCount: number;
  role: MemberRole | null;
  tags: homePage_homePage_organizations_tags[];
  translations: homePage_homePage_organizations_translations[];
}

export interface homePage_homePage_projects_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface homePage_homePage_projects_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: homePage_homePage_projects_tags_translations[];
}

export interface homePage_homePage_projects_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface homePage_homePage_projects {
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
  tags: homePage_homePage_projects_tags[];
  translations: homePage_homePage_projects_translations[];
}

export interface homePage_homePage_routines_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface homePage_homePage_routines_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: homePage_homePage_routines_tags_translations[];
}

export interface homePage_homePage_routines_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface homePage_homePage_routines {
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
  tags: homePage_homePage_routines_tags[];
  translations: homePage_homePage_routines_translations[];
  version: string | null;
}

export interface homePage_homePage_standards_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface homePage_homePage_standards_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: homePage_homePage_standards_tags_translations[];
}

export interface homePage_homePage_standards_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface homePage_homePage_standards {
  __typename: "Standard";
  id: string;
  commentsCount: number;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  reportsCount: number;
  role: MemberRole | null;
  tags: homePage_homePage_standards_tags[];
  translations: homePage_homePage_standards_translations[];
  type: string;
}

export interface homePage_homePage_users {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
  reportsCount: number;
}

export interface homePage_homePage {
  __typename: "HomePageResult";
  organizations: homePage_homePage_organizations[];
  projects: homePage_homePage_projects[];
  routines: homePage_homePage_routines[];
  standards: homePage_homePage_standards[];
  users: homePage_homePage_users[];
}

export interface homePage {
  homePage: homePage_homePage;
}

export interface homePageVariables {
  input: HomePageInput;
}
