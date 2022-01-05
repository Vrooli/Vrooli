/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StandardType, NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineFields
// ====================================================

export interface deepRoutineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_inputs_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_inputs_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: deepRoutineFields_inputs_routine_tags[];
}

export interface deepRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_inputs_standard {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: deepRoutineFields_inputs_standard_tags[];
}

export interface deepRoutineFields_inputs {
  __typename: "RoutineInputItem";
  routine: deepRoutineFields_inputs_routine;
  standard: deepRoutineFields_inputs_standard;
}

export interface deepRoutineFields_outputs_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_outputs_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: deepRoutineFields_outputs_routine_tags[];
}

export interface deepRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_outputs_standard {
  __typename: "Standard";
  id: string;
  name: string;
  description: string | null;
  type: StandardType;
  schema: string;
  default: string | null;
  isFile: boolean;
  created_at: any;
  tags: deepRoutineFields_outputs_standard_tags[];
}

export interface deepRoutineFields_outputs {
  __typename: "RoutineOutputItem";
  routine: deepRoutineFields_outputs_routine;
  standard: deepRoutineFields_outputs_standard;
}

export interface deepRoutineFields_nodes_data_NodeCombine {
  __typename: "NodeCombine";
  id: string;
  from: string[];
  to: string;
}

export interface deepRoutineFields_nodes_data_NodeDecision_decisions_when {
  __typename: "NodeDecisionItemCase";
  id: string;
  condition: string;
}

export interface deepRoutineFields_nodes_data_NodeDecision_decisions {
  __typename: "NodeDecisionItem";
  id: string;
  title: string;
  description: string | null;
  toId: string | null;
  when: (deepRoutineFields_nodes_data_NodeDecision_decisions_when | null)[];
}

export interface deepRoutineFields_nodes_data_NodeDecision {
  __typename: "NodeDecision";
  id: string;
  decisions: deepRoutineFields_nodes_data_NodeDecision_decisions[];
}

export interface deepRoutineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface deepRoutineFields_nodes_data_NodeLoop {
  __typename: "NodeLoop";
  id: string;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine_tags[];
}

export interface deepRoutineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  title: string;
  description: string | null;
  isOptional: boolean;
  routine: deepRoutineFields_nodes_data_NodeRoutineList_routines_routine | null;
}

export interface deepRoutineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOrdered: boolean;
  isOptional: boolean;
  routines: deepRoutineFields_nodes_data_NodeRoutineList_routines[];
}

export interface deepRoutineFields_nodes_data_NodeRedirect {
  __typename: "NodeRedirect";
  id: string;
}

export interface deepRoutineFields_nodes_data_NodeStart {
  __typename: "NodeStart";
  id: string;
}

export type deepRoutineFields_nodes_data = deepRoutineFields_nodes_data_NodeCombine | deepRoutineFields_nodes_data_NodeDecision | deepRoutineFields_nodes_data_NodeEnd | deepRoutineFields_nodes_data_NodeLoop | deepRoutineFields_nodes_data_NodeRoutineList | deepRoutineFields_nodes_data_NodeRedirect | deepRoutineFields_nodes_data_NodeStart;

export interface deepRoutineFields_nodes {
  __typename: "Node";
  id: string;
  created_at: any;
  updated_at: any;
  routineId: string;
  title: string;
  description: string | null;
  type: NodeType;
  data: deepRoutineFields_nodes_data | null;
  previous: string | null;
  next: string | null;
}

export interface deepRoutineFields {
  __typename: "Routine";
  id: string;
  version: string | null;
  title: string | null;
  description: string | null;
  created_at: any;
  isAutomatable: boolean | null;
  tags: deepRoutineFields_tags[];
  instructions: string | null;
  inputs: deepRoutineFields_inputs[];
  outputs: deepRoutineFields_outputs[];
  nodes: deepRoutineFields_nodes[];
}
