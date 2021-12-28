/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardsQueryInput, StandardType } from "./globalTypes";

// ====================================================
// GraphQL query operation: standards
// ====================================================

export interface standards_standards {
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

export interface standards {
  standards: standards_standards[];
}

export interface standardsVariables {
  input: StandardsQueryInput;
}
