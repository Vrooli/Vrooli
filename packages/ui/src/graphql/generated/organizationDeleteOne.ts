/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: organizationDeleteOne
// ====================================================

export interface organizationDeleteOne_organizationDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface organizationDeleteOne {
  organizationDeleteOne: organizationDeleteOne_organizationDeleteOne;
}

export interface organizationDeleteOneVariables {
  input: DeleteOneInput;
}
