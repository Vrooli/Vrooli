/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { OrganizationInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationUpdate
// ====================================================

export interface organizationUpdate_organizationUpdate {
  __typename: "Organization";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
}

export interface organizationUpdate {
  organizationUpdate: organizationUpdate_organizationUpdate;
}

export interface organizationUpdateVariables {
  input: OrganizationInput;
}
