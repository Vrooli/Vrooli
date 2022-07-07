/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCreateInput, NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCreate
// ====================================================

export interface routineCreate_routineCreate_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_inputs_standard_translations[];
  version: string;
}

export interface routineCreate_routineCreate_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineCreate_routineCreate_inputs_translations[];
  standard: routineCreate_routineCreate_inputs_standard | null;
}

export interface routineCreate_routineCreate_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineCreate_routineCreate_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: routineCreate_routineCreate_nodeLinks_whens_translations[];
}

export interface routineCreate_routineCreate_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: routineCreate_routineCreate_nodeLinks_whens[];
}

export interface routineCreate_routineCreate_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner = routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_Organization | routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isInternal: boolean | null;
  inputs: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  role: MemberRole | null;
  outputs: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_owner | null;
  resourceLists: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routine;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines[];
}

export type routineCreate_routineCreate_nodes_data = routineCreate_routineCreate_nodes_data_NodeEnd | routineCreate_routineCreate_nodes_data_NodeRoutineList;

export interface routineCreate_routineCreate_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineCreate_routineCreate_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineCreate_routineCreate_nodes_loop_whiles_translations[];
}

export interface routineCreate_routineCreate_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineCreate_routineCreate_nodes_loop_whiles[];
}

export interface routineCreate_routineCreate_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineCreate_routineCreate_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineCreate_routineCreate_nodes_data | null;
  loop: routineCreate_routineCreate_nodes_loop | null;
  translations: routineCreate_routineCreate_nodes_translations[];
}

export interface routineCreate_routineCreate_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_outputs_standard_translations[];
  version: string;
}

export interface routineCreate_routineCreate_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineCreate_routineCreate_outputs_translations[];
  standard: routineCreate_routineCreate_outputs_standard | null;
}

export interface routineCreate_routineCreate_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineCreate_routineCreate_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineCreate_routineCreate_owner_Organization_translations[];
}

export interface routineCreate_routineCreate_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineCreate_routineCreate_owner = routineCreate_routineCreate_owner_Organization | routineCreate_routineCreate_owner_User;

export interface routineCreate_routineCreate_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface routineCreate_routineCreate_parent {
  __typename: "Routine";
  id: string;
  translations: routineCreate_routineCreate_parent_translations[];
}

export interface routineCreate_routineCreate_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineCreate_routineCreate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineCreate_routineCreate_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineCreate_routineCreate_resourceLists_resources_translations[];
}

export interface routineCreate_routineCreate_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineCreate_routineCreate_resourceLists_translations[];
  resources: routineCreate_routineCreate_resourceLists_resources[];
}

export interface routineCreate_routineCreate_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineCreate_routineCreate_runs_steps {
  __typename: "RunStep";
  id: string;
  order: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStepStatus;
  step: number[];
  node: routineCreate_routineCreate_runs_steps_node | null;
}

export interface routineCreate_routineCreate_runs {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineCreate_routineCreate_runs_steps[];
}

export interface routineCreate_routineCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: routineCreate_routineCreate_tags_translations[];
}

export interface routineCreate_routineCreate_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface routineCreate_routineCreate {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: routineCreate_routineCreate_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: routineCreate_routineCreate_nodeLinks[];
  nodes: routineCreate_routineCreate_nodes[];
  outputs: routineCreate_routineCreate_outputs[];
  owner: routineCreate_routineCreate_owner | null;
  parent: routineCreate_routineCreate_parent | null;
  resourceLists: routineCreate_routineCreate_resourceLists[];
  runs: routineCreate_routineCreate_runs[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: routineCreate_routineCreate_tags[];
  translations: routineCreate_routineCreate_translations[];
  updated_at: any;
  version: string | null;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
