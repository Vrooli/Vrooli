/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: standardFields
// ====================================================

export interface standardFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: standardFields_tags_translations[];
}

export interface standardFields_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardFields_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface standardFields_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: standardFields_creator_Organization_translations[];
}

export interface standardFields_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type standardFields_creator = standardFields_creator_Organization | standardFields_creator_User;

export interface standardFields {
  __typename: "Standard";
  id: string;
  name: string;
  role: MemberRole | null;
  type: string;
  props: string;
  yup: string | null;
  default: string | null;
  created_at: any;
  tags: standardFields_tags[];
  translations: standardFields_translations[];
  creator: standardFields_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
  version: string;
}
