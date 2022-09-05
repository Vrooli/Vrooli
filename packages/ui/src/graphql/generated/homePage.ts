/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { HomePageInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: homePage
// ====================================================

export interface homePage_homePage_organizations_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

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
  isPrivate: boolean;
  isStarred: boolean;
  membersCount: number;
  reportsCount: number;
  permissionsOrganization: homePage_homePage_organizations_permissionsOrganization | null;
  tags: homePage_homePage_organizations_tags[];
  translations: homePage_homePage_organizations_translations[];
}

export interface homePage_homePage_projects_permissionsProject {
  __typename: "ProjectPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
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
  score: number;
  stars: number;
  isComplete: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  reportsCount: number;
  permissionsProject: homePage_homePage_projects_permissionsProject;
  tags: homePage_homePage_projects_tags[];
  translations: homePage_homePage_projects_translations[];
}

export interface homePage_homePage_routines_permissionsRoutine {
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
  permissionsRoutine: homePage_homePage_routines_permissionsRoutine;
  tags: homePage_homePage_routines_tags[];
  translations: homePage_homePage_routines_translations[];
  version: string;
  versionGroupId: string;
}

export interface homePage_homePage_standards_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
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
  jsonVariable: string | null;
}

export interface homePage_homePage_standards {
  __typename: "Standard";
  id: string;
  commentsCount: number;
  default: string | null;
  score: number;
  stars: number;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  props: string;
  reportsCount: number;
  permissionsStandard: homePage_homePage_standards_permissionsStandard;
  tags: homePage_homePage_standards_tags[];
  translations: homePage_homePage_standards_translations[];
  type: string;
  version: string;
  versionGroupId: string;
  yup: string | null;
}

export interface homePage_homePage_users_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface homePage_homePage_users {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
  reportsCount: number;
  translations: homePage_homePage_users_translations[];
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
