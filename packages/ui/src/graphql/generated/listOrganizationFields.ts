/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: listOrganizationFields
// ====================================================

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
  handle: string | null;
  stars: number;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  tags: listOrganizationFields_tags[];
  translations: listOrganizationFields_translations[];
}
