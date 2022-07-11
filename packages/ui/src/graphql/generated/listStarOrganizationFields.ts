/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: listStarOrganizationFields
// ====================================================

export interface listStarOrganizationFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listStarOrganizationFields_tags {
  __typename: "Tag";
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
  isStarred: boolean;
  role: MemberRole | null;
  tags: listStarOrganizationFields_tags[];
  translations: listStarOrganizationFields_translations[];
}
