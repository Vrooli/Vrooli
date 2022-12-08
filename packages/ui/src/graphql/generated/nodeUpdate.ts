/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeUpdateInput, NodeType } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: nodeUpdate
// ====================================================

export interface nodeUpdate_nodeUpdate_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  simplicity: number;
  permissionsRoutine: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  tags: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_routineVersion;
  translations: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines_translations[];
}

export interface nodeUpdate_nodeUpdate_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: nodeUpdate_nodeUpdate_data_NodeRoutineList_routines[];
}

export type nodeUpdate_nodeUpdate_data = nodeUpdate_nodeUpdate_data_NodeEnd | nodeUpdate_nodeUpdate_data_NodeRoutineList;

export interface nodeUpdate_nodeUpdate_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface nodeUpdate_nodeUpdate_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: nodeUpdate_nodeUpdate_loop_whiles_translations[];
}

export interface nodeUpdate_nodeUpdate_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: nodeUpdate_nodeUpdate_loop_whiles[];
}

export interface nodeUpdate_nodeUpdate_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface nodeUpdate_nodeUpdate {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: nodeUpdate_nodeUpdate_data | null;
  loop: nodeUpdate_nodeUpdate_loop | null;
  translations: nodeUpdate_nodeUpdate_translations[];
}

export interface nodeUpdate {
  nodeUpdate: nodeUpdate_nodeUpdate;
}

export interface nodeUpdateVariables {
  input: NodeUpdateInput;
}
