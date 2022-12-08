/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: nodeFields
// ====================================================

export interface nodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface nodeFields_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface nodeFields_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeFields_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: nodeFields_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface nodeFields_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
}

export interface nodeFields_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: nodeFields_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  tags: nodeFields_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: nodeFields_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface nodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface nodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: nodeFields_data_NodeRoutineList_routines_routineVersion;
  translations: nodeFields_data_NodeRoutineList_routines_translations[];
}

export interface nodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeFields_data_NodeRoutineList_routines[];
}

export type nodeFields_data = nodeFields_data_NodeEnd | nodeFields_data_NodeRoutineList;

export interface nodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface nodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: nodeFields_loop_whiles_translations[];
}

export interface nodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: nodeFields_loop_whiles[];
}

export interface nodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface nodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: nodeFields_data | null;
  loop: nodeFields_loop | null;
  translations: nodeFields_translations[];
}
