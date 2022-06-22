/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: listStandardFields
// ====================================================

export interface listStandardFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStandardFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStandardFields_tags_translations[];
}

export interface listStandardFields_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStandardFields {
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
  tags: listStandardFields_tags[];
  translations: listStandardFields_translations[];
}
