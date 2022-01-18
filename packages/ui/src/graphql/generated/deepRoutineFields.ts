/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType } from "./globalTypes";

// ====================================================
// GraphQL fragment: deepRoutineFields
// ====================================================

export interface deepRoutineFields_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface deepRoutineFields_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_inputs_standard_tags[];
}

export interface deepRoutineFields_inputs {
  __typename: "RoutineInputItem";
  id: string;
  standard: deepRoutineFields_inputs_standard;
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
  description: string | null;
  title: string;
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
  stars: number;
  score: number;
  isUpvoted: boolean;
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
  stars: number;
  score: number;
  isUpvoted: boolean;
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
  isOptional: boolean;
  isOrdered: boolean;
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
  description: string | null;
  next: string | null;
  previous: string | null;
  title: string;
  type: NodeType;
  updated_at: any;
  data: deepRoutineFields_nodes_data | null;
}

export interface deepRoutineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  name: string;
}

export interface deepRoutineFields_owner_User {
  __typename: "User";
  id: string;
  username: string | null;
}

export type deepRoutineFields_owner = deepRoutineFields_owner_Organization | deepRoutineFields_owner_User;

export interface deepRoutineFields_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface deepRoutineFields_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  description: string | null;
  isFile: boolean;
  name: string;
  schema: string;
  tags: deepRoutineFields_outputs_standard_tags[];
}

export interface deepRoutineFields_outputs {
  __typename: "RoutineOutputItem";
  id: string;
  standard: deepRoutineFields_outputs_standard;
}

export interface deepRoutineFields_parent {
  __typename: "Routine";
  id: string;
  title: string | null;
}

export interface deepRoutineFields_contextualResources {
  __typename: "Resource";
  id: string;
  title: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface deepRoutineFields_externalResources {
  __typename: "Resource";
  id: string;
  title: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface deepRoutineFields_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  score: number;
  isUpvoted: boolean;
}

export interface deepRoutineFields {
  __typename: "Routine";
  id: string;
  created_at: any;
  instructions: string | null;
  isAutomatable: boolean | null;
  title: string | null;
  description: string | null;
  updated_at: any;
  version: string | null;
  stars: number;
  inputs: deepRoutineFields_inputs[];
  nodes: deepRoutineFields_nodes[];
  owner: deepRoutineFields_owner | null;
  outputs: deepRoutineFields_outputs[];
  parent: deepRoutineFields_parent | null;
  contextualResources: deepRoutineFields_contextualResources[];
  externalResources: deepRoutineFields_externalResources[];
  tags: deepRoutineFields_tags[];
}
