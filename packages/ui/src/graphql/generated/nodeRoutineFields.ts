/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeRoutineFields
// ====================================================

export interface nodeRoutineFields_tags {
  __typename: "Tag";
  id: string;
  description: string | null;
  tag: string;
}

export interface nodeRoutineFields {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  role: MemberRole | null;
  tags: nodeRoutineFields_tags[];
}
