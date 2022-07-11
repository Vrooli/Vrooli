/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: listProjectFields
// ====================================================

export interface listProjectFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listProjectFields_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listProjectFields_tags_translations[];
}

export interface listProjectFields_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface listProjectFields {
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
  tags: listProjectFields_tags[];
  translations: listProjectFields_translations[];
}
