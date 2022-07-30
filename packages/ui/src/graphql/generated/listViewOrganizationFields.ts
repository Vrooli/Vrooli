/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: listViewOrganizationFields
// ====================================================

export interface listViewOrganizationFields_permissionsOrganization {
  __typename: "OrganizationPermission";
  canAddMembers: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  isMember: boolean;
}

export interface listViewOrganizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewOrganizationFields_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewOrganizationFields_tags_translations[];
}

export interface listViewOrganizationFields_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface listViewOrganizationFields {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isPrivate: boolean;
  isStarred: boolean;
  permissionsOrganization: listViewOrganizationFields_permissionsOrganization | null;
  tags: listViewOrganizationFields_tags[];
  translations: listViewOrganizationFields_translations[];
}
