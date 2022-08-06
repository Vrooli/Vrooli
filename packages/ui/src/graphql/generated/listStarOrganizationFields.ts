/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listStarOrganizationFields
// ====================================================

export interface listStarOrganizationFields_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface listStarOrganizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarOrganizationFields_tags {
  __typename: "Tag";
  id: string;
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listStarOrganizationFields_tags_translations[];
}

export interface listStarOrganizationFields_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface listStarOrganizationFields {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: listStarOrganizationFields_permissionsOrganization | null;
  tags: listStarOrganizationFields_tags[];
  translations: listStarOrganizationFields_translations[];
}
