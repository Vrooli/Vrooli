/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationsQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: organizations
// ====================================================

export interface organizations_organizations {
  __typename: "Organization";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
}

export interface organizations {
  organizations: organizations_organizations[];
}

export interface organizationsVariables {
  input: OrganizationsQueryInput;
}
