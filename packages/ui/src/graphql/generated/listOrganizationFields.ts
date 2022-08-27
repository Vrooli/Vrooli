/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listOrganizationFields
// ====================================================

export interface listOrganizationFields_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface listOrganizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listOrganizationFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listOrganizationFields_tags_translations[];
}

export interface listOrganizationFields_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface listOrganizationFields {
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
  permissionsOrganization: listOrganizationFields_permissionsOrganization | null;
  tags: listOrganizationFields_tags[];
  translations: listOrganizationFields_translations[];
}
