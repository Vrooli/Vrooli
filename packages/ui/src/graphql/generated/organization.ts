/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL query operation: organization
// ====================================================

export interface organization_organization_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface organization_organization {
  __typename: "Organization";
  id: string;
  bio: string | null;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  stars: number;
  tags: organization_organization_tags[];
}

export interface organization {
  organization: organization_organization | null;
}

export interface organizationVariables {
  input: FindByIdInput;
}
