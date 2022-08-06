/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarStandardFields
// ====================================================

export interface listStarStandardFields_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarStandardFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarStandardFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarStandardFields_tags_translations[];
}

export interface listStarStandardFields_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarStandardFields {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: listStarStandardFields_permissionsStandard;
  tags: listStarStandardFields_tags[];
  translations: listStarStandardFields_translations[];
  version: string;
  versionGroupId: string;
}
