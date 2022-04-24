/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ForYouPageInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: forYouPage
// ====================================================

export interface forYouPage_forYouPage_activeRoutines_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_activeRoutines_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_activeRoutines_tags_translations[];
}

export interface forYouPage_forYouPage_activeRoutines_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_activeRoutines {
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
  tags: forYouPage_forYouPage_activeRoutines_tags[];
  translations: forYouPage_forYouPage_activeRoutines_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_completedRoutines_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_completedRoutines_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_completedRoutines_tags_translations[];
}

export interface forYouPage_forYouPage_completedRoutines_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_completedRoutines {
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
  tags: forYouPage_forYouPage_completedRoutines_tags[];
  translations: forYouPage_forYouPage_completedRoutines_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_recent_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recent_Project_tags_translations[];
}

export interface forYouPage_forYouPage_recent_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: forYouPage_forYouPage_recent_Project_tags[];
  translations: forYouPage_forYouPage_recent_Project_translations[];
}

export interface forYouPage_forYouPage_recent_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recent_Organization_tags_translations[];
}

export interface forYouPage_forYouPage_recent_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface forYouPage_forYouPage_recent_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recent_Organization_tags[];
  translations: forYouPage_forYouPage_recent_Organization_translations[];
}

export interface forYouPage_forYouPage_recent_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recent_Routine_tags_translations[];
}

export interface forYouPage_forYouPage_recent_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_recent_Routine {
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
  tags: forYouPage_forYouPage_recent_Routine_tags[];
  translations: forYouPage_forYouPage_recent_Routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_recent_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_recent_Standard_tags_translations[];
}

export interface forYouPage_forYouPage_recent_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_recent_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_recent_Standard_tags[];
  translations: forYouPage_forYouPage_recent_Standard_translations[];
}

export interface forYouPage_forYouPage_recent_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type forYouPage_forYouPage_recent = forYouPage_forYouPage_recent_Project | forYouPage_forYouPage_recent_Organization | forYouPage_forYouPage_recent_Routine | forYouPage_forYouPage_recent_Standard | forYouPage_forYouPage_recent_User;

export interface forYouPage_forYouPage_starred_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Project_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_starred_Project_tags_translations[];
}

export interface forYouPage_forYouPage_starred_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: forYouPage_forYouPage_starred_Project_tags[];
  translations: forYouPage_forYouPage_starred_Project_translations[];
}

export interface forYouPage_forYouPage_starred_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Organization_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_starred_Organization_tags_translations[];
}

export interface forYouPage_forYouPage_starred_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface forYouPage_forYouPage_starred_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_starred_Organization_tags[];
  translations: forYouPage_forYouPage_starred_Organization_translations[];
}

export interface forYouPage_forYouPage_starred_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Routine_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_starred_Routine_tags_translations[];
}

export interface forYouPage_forYouPage_starred_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface forYouPage_forYouPage_starred_Routine {
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
  tags: forYouPage_forYouPage_starred_Routine_tags[];
  translations: forYouPage_forYouPage_starred_Routine_translations[];
  version: string | null;
}

export interface forYouPage_forYouPage_starred_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Standard_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: forYouPage_forYouPage_starred_Standard_tags_translations[];
}

export interface forYouPage_forYouPage_starred_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface forYouPage_forYouPage_starred_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: forYouPage_forYouPage_starred_Standard_tags[];
  translations: forYouPage_forYouPage_starred_Standard_translations[];
}

export interface forYouPage_forYouPage_starred_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type forYouPage_forYouPage_starred = forYouPage_forYouPage_starred_Project | forYouPage_forYouPage_starred_Organization | forYouPage_forYouPage_starred_Routine | forYouPage_forYouPage_starred_Standard | forYouPage_forYouPage_starred_User;

export interface forYouPage_forYouPage {
  __typename: "ForYouPageResult";
  activeRoutines: forYouPage_forYouPage_activeRoutines[];
  completedRoutines: forYouPage_forYouPage_completedRoutines[];
  recent: forYouPage_forYouPage_recent[];
  starred: forYouPage_forYouPage_starred[];
}

export interface forYouPage {
  forYouPage: forYouPage_forYouPage;
}

export interface forYouPageVariables {
  input: ForYouPageInput;
}
