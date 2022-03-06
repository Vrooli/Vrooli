/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: organizationFields
// ====================================================

export interface organizationFields_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface organizationFields {
  __typename: "Organization";
  id: string;
  bio: string | null;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  stars: number;
  tags: organizationFields_tags[];
}
