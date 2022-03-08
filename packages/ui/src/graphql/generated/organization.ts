/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput, MemberRole, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL query operation: organization
// ====================================================

export interface organization_organization_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface organization_organization_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: organization_organization_resources_translations[];
}

export interface organization_organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface organization_organization_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: organization_organization_tags_translations[];
}

export interface organization_organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  bio: string | null;
  name: string;
}

export interface organization_organization {
  __typename: "Organization";
  id: string;
  created_at: any;
  isOpenToNewMembers: boolean;
  isStarred: boolean;
  role: MemberRole | null;
  stars: number;
  resources: organization_organization_resources[];
  tags: organization_organization_tags[];
  translations: organization_organization_translations[];
}

export interface organization {
  organization: organization_organization | null;
}

export interface organizationVariables {
  input: FindByIdInput;
}
