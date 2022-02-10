/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationCreateInput, MemberRole } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationCreate
// ====================================================

export interface organizationCreate_organizationCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean | null;
}

export interface organizationCreate_organizationCreate {
  __typename: "Organization";
  id: string;
  name: string;
  bio: string | null;
  created_at: any;
  tags: organizationCreate_organizationCreate_tags[];
  stars: number;
  isStarred: boolean | null;
  role: MemberRole | null;
}

export interface organizationCreate {
  organizationCreate: organizationCreate_organizationCreate;
}

export interface organizationCreateVariables {
  input: OrganizationCreateInput;
}
