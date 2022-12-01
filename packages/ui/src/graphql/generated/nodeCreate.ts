/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeCreateInput, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeCreate
// ====================================================

export interface nodeCreate_nodeCreate_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  tags: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: nodeCreate_nodeCreate_data_NodeRoutineList_routines_routineVersion;
  translations: nodeCreate_nodeCreate_data_NodeRoutineList_routines_translations[];
}

export interface nodeCreate_nodeCreate_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeCreate_nodeCreate_data_NodeRoutineList_routines[];
}

export type nodeCreate_nodeCreate_data = nodeCreate_nodeCreate_data_NodeEnd | nodeCreate_nodeCreate_data_NodeRoutineList;

export interface nodeCreate_nodeCreate_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface nodeCreate_nodeCreate_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: nodeCreate_nodeCreate_loop_whiles_translations[];
}

export interface nodeCreate_nodeCreate_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: nodeCreate_nodeCreate_loop_whiles[];
}

export interface nodeCreate_nodeCreate_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface nodeCreate_nodeCreate {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: nodeCreate_nodeCreate_data | null;
  loop: nodeCreate_nodeCreate_loop | null;
  translations: nodeCreate_nodeCreate_translations[];
}

export interface nodeCreate {
  nodeCreate: nodeCreate_nodeCreate;
}

export interface nodeCreateVariables {
  input: NodeCreateInput;
}
