/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listViewStandardFields
// ====================================================

export interface listViewStandardFields_permissionsStandard {
  __typename: "StandardPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listViewStandardFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewStandardFields_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewStandardFields_tags_translations[];
}

export interface listViewStandardFields_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface listViewStandardFields {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isDeleted: boolean;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  permissionsStandard: listViewStandardFields_permissionsStandard;
  tags: listViewStandardFields_tags[];
  translations: listViewStandardFields_translations[];
}
