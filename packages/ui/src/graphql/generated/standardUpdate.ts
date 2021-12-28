/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardInput, StandardType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: standardUpdate
// ====================================================

export interface standardUpdate_standardUpdate {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
}

export interface standardUpdate {
  standardUpdate: standardUpdate_standardUpdate;
}

export interface standardUpdateVariables {
  input: StandardInput;
}
