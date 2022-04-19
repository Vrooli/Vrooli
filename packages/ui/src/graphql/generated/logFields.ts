/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { LogType } from "./globalTypes";

// ====================================================
// GraphQL fragment: logFields
// ====================================================

export interface logFields {
  __typename: "Log";
  id: string;
  timestamp: any;
  action: LogType;
  object1Type: string | null;
  object1Id: string | null;
  object2Type: string | null;
  object2Id: string | null;
  data: string | null;
}
