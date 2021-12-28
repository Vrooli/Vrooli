/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: organization
// ====================================================

export interface organization_organization {
  __typename: "Organization";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
}

export interface organization {
  organization: organization_organization | null;
}

export interface organizationVariables {
  input: FindByIdInput;
}
