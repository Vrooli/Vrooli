/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutinesQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: routines
// ====================================================

export interface routines_routines {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
}

export interface routines {
  routines: routines_routines[];
}

export interface routinesVariables {
  input: RoutinesQueryInput;
}
