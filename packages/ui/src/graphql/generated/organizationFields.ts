/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: organizationFields
// ====================================================

export interface organizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organizationFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: organizationFields_tags_translations[];
}

export interface organizationFields_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organizationFields {
  __typename: "Organization";
  id: string;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  tags: organizationFields_tags[];
  translations: organizationFields_translations[];
}
