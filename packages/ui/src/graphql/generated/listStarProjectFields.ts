/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarProjectFields
// ====================================================

export interface listStarProjectFields_permissionsProject {
  __typename: "ProjectPermission";
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canVote: boolean;
}

export interface listStarProjectFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarProjectFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarProjectFields_tags_translations[];
}

export interface listStarProjectFields_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface listStarProjectFields {
  __typename: "Project";
  id: string;
  handle: string | null;
  score: number;
  stars: number;
  isPrivate: boolean;
  isUpvoted: boolean | null;
  isStarred: boolean;
  permissionsProject: listStarProjectFields_permissionsProject;
  tags: listStarProjectFields_tags[];
  translations: listStarProjectFields_translations[];
}
