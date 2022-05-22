/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardCreate
// ====================================================

export interface standardCreate_standardCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardCreate_standardCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: standardCreate_standardCreate_tags_translations[];
}

export interface standardCreate_standardCreate_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface standardCreate_standardCreate_creator_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface standardCreate_standardCreate_creator_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: standardCreate_standardCreate_creator_Organization_translations[];
}

export interface standardCreate_standardCreate_creator_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type standardCreate_standardCreate_creator = standardCreate_standardCreate_creator_Organization | standardCreate_standardCreate_creator_User;

export interface standardCreate_standardCreate {
  __typename: "Standard";
  id: string;
  name: string;
  role: MemberRole | null;
  type: string;
  props: string;
  yup: string | null;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: standardCreate_standardCreate_tags[];
  translations: standardCreate_standardCreate_translations[];
  creator: standardCreate_standardCreate_creator | null;
  stars: number;
  isStarred: boolean;
  score: number;
  isUpvoted: boolean | null;
}

export interface standardCreate {
  standardCreate: standardCreate_standardCreate;
}

export interface standardCreateVariables {
  input: StandardCreateInput;
}
