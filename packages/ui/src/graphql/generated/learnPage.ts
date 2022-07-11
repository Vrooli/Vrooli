/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: learnPage
// ====================================================

export interface learnPage_learnPage_courses_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_courses_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: learnPage_learnPage_courses_tags_translations[];
}

export interface learnPage_learnPage_courses_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface learnPage_learnPage_courses {
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
  tags: learnPage_learnPage_courses_tags[];
  translations: learnPage_learnPage_courses_translations[];
}

export interface learnPage_learnPage_tutorials_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface learnPage_learnPage_tutorials_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: learnPage_learnPage_tutorials_tags_translations[];
}

export interface learnPage_learnPage_tutorials_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface learnPage_learnPage_tutorials {
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
  tags: learnPage_learnPage_tutorials_tags[];
  translations: learnPage_learnPage_tutorials_translations[];
  version: string | null;
}

export interface learnPage_learnPage {
  __typename: "LearnPageResult";
  courses: learnPage_learnPage_courses[];
  tutorials: learnPage_learnPage_tutorials[];
}

export interface learnPage {
  learnPage: learnPage_learnPage;
}
