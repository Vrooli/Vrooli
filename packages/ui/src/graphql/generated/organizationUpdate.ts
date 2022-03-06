/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationUpdateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationUpdate
// ====================================================

export interface organizationUpdate_organizationUpdate_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface organizationUpdate_organizationUpdate {
  __typename: "Organization";
  id: string;
  bio: string | null;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  stars: number;
  tags: organizationUpdate_organizationUpdate_tags[];
}

export interface organizationUpdate {
  organizationUpdate: organizationUpdate_organizationUpdate;
}

export interface organizationUpdateVariables {
  input: OrganizationUpdateInput;
}
