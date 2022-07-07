/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RunCompleteInput, RunStatus, NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: runComplete
// ====================================================

export interface runComplete_runComplete_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_inputs_standard_tags_translations[];
}

export interface runComplete_runComplete_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runComplete_runComplete_routine_inputs_standard_tags[];
  translations: runComplete_runComplete_routine_inputs_standard_translations[];
  version: string;
}

export interface runComplete_runComplete_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runComplete_runComplete_routine_inputs_translations[];
  standard: runComplete_runComplete_routine_inputs_standard | null;
}

export interface runComplete_runComplete_routine_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runComplete_runComplete_routine_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: runComplete_runComplete_routine_nodeLinks_whens_translations[];
}

export interface runComplete_runComplete_routine_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: runComplete_runComplete_routine_nodeLinks_whens[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_tags[];
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_tags[];
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner = runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_Organization | runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_inputs[];
  isComplete: boolean;
  isInternal: boolean | null;
  nodesCount: number | null;
  role: MemberRole | null;
  outputs: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_owner | null;
  resourceLists: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_routine;
  translations: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines_translations[];
}

export interface runComplete_runComplete_routine_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runComplete_runComplete_routine_nodes_data_NodeRoutineList_routines[];
}

export type runComplete_runComplete_routine_nodes_data = runComplete_runComplete_routine_nodes_data_NodeEnd | runComplete_runComplete_routine_nodes_data_NodeRoutineList;

export interface runComplete_runComplete_routine_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runComplete_runComplete_routine_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runComplete_runComplete_routine_nodes_loop_whiles_translations[];
}

export interface runComplete_runComplete_routine_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runComplete_runComplete_routine_nodes_loop_whiles[];
}

export interface runComplete_runComplete_routine_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface runComplete_runComplete_routine_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runComplete_runComplete_routine_nodes_data | null;
  loop: runComplete_runComplete_routine_nodes_loop | null;
  translations: runComplete_runComplete_routine_nodes_translations[];
}

export interface runComplete_runComplete_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_outputs_standard_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_outputs_standard_tags_translations[];
}

export interface runComplete_runComplete_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  tags: runComplete_runComplete_routine_outputs_standard_tags[];
  translations: runComplete_runComplete_routine_outputs_standard_translations[];
  version: string;
}

export interface runComplete_runComplete_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runComplete_runComplete_routine_outputs_translations[];
  standard: runComplete_runComplete_routine_outputs_standard | null;
}

export interface runComplete_runComplete_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runComplete_runComplete_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runComplete_runComplete_routine_owner_Organization_translations[];
}

export interface runComplete_runComplete_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runComplete_runComplete_routine_owner = runComplete_runComplete_routine_owner_Organization | runComplete_runComplete_routine_owner_User;

export interface runComplete_runComplete_routine_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface runComplete_runComplete_routine_parent {
  __typename: "Routine";
  id: string;
  translations: runComplete_runComplete_routine_parent_translations[];
}

export interface runComplete_runComplete_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runComplete_runComplete_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface runComplete_runComplete_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runComplete_runComplete_routine_resourceLists_resources_translations[];
}

export interface runComplete_runComplete_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runComplete_runComplete_routine_resourceLists_translations[];
  resources: runComplete_runComplete_routine_resourceLists_resources[];
}

export interface runComplete_runComplete_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runComplete_runComplete_routine_tags {
  __typename: "Tag";
  id: string;
  tag: string;
  translations: runComplete_runComplete_routine_tags_translations[];
}

export interface runComplete_runComplete_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface runComplete_runComplete_routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: runComplete_runComplete_routine_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isInternal: boolean | null;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: runComplete_runComplete_routine_nodeLinks[];
  nodes: runComplete_runComplete_routine_nodes[];
  outputs: runComplete_runComplete_routine_outputs[];
  owner: runComplete_runComplete_routine_owner | null;
  parent: runComplete_runComplete_routine_parent | null;
  resourceLists: runComplete_runComplete_routine_resourceLists[];
  score: number;
  simplicity: number;
  stars: number;
  role: MemberRole | null;
  tags: runComplete_runComplete_routine_tags[];
  translations: runComplete_runComplete_routine_translations[];
  updated_at: any;
  version: string | null;
}

export interface runComplete_runComplete_steps_node {
  __typename: "Node";
  id: string;
}

export interface runComplete_runComplete_steps {
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
  node: runComplete_runComplete_steps_node | null;
}

export interface runComplete_runComplete {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  routine: runComplete_runComplete_routine | null;
  steps: runComplete_runComplete_steps[];
}

export interface runComplete {
  runComplete: runComplete_runComplete;
}

export interface runCompleteVariables {
  input: RunCompleteInput;
}
