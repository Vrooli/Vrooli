/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStandardFields
// ====================================================

export interface listStandardFields_permissionsStandard {
  __typename: "StandardPermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

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
  jsonVariable: string | null;
}

export interface listStandardFields {
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
  permissionsStandard: listStandardFields_permissionsStandard;
  tags: listStandardFields_tags[];
  translations: listStandardFields_translations[];
  type: string;
  version: string;
  versionGroupId: string;
  yup: string | null;
}
