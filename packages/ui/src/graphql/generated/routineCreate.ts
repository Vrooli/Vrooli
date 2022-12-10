/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { RoutineCreateInput, NodeType, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunRoutineStepStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: routineCreate
// ====================================================

export interface routineCreate_routineCreate_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineCreate_routineCreate_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineCreate_routineCreate_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_inputs_standard_translations[];
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
  name: string;
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

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner = routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_Organization | routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
  instructions: string;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_inputs[];
  nodesCount: number | null;
  outputs: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface routineCreate_routineCreate_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: routineCreate_routineCreate_nodes_data_NodeRoutineList_routines_routineVersion;
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
  name: string;
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
  name: string;
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
  helpText: string | null;
}

export interface routineCreate_routineCreate_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineCreate_routineCreate_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isDeleted: boolean;
  isInternal: boolean;
  isPrivate: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineCreate_routineCreate_outputs_standard_translations[];
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
  name: string;
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
  name: string | null;
}

export interface routineCreate_routineCreate_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
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

export interface routineCreate_routineCreate_runs_inputs_input {
  __typename: "InputItem";
  id: string;
}

export interface routineCreate_routineCreate_runs_inputs {
  __typename: "RunRoutineInput";
  id: string;
  data: string;
  input: routineCreate_routineCreate_runs_inputs_input;
}

export interface routineCreate_routineCreate_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineCreate_routineCreate_runs_steps {
  __typename: "RunRoutineStep";
  id: string;
  order: number;
  contextSwitches: number;
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunRoutineStepStatus;
  step: number[];
  node: routineCreate_routineCreate_runs_steps_node | null;
}

export interface routineCreate_routineCreate_runs {
  __typename: "RunRoutine";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  inputs: routineCreate_routineCreate_runs_inputs[];
  startedAt: any | null;
  timeElapsed: number | null;
  completedAt: any | null;
  name: string;
  status: RunStatus;
  steps: routineCreate_routineCreate_runs_steps[];
}

export interface routineCreate_routineCreate_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineCreate_routineCreate_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineCreate_routineCreate_tags {
  __typename: "Tag";
  tag: string;
  translations: routineCreate_routineCreate_tags_translations[];
}

export interface routineCreate_routineCreate_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  name: string;
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
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: routineCreate_routineCreate_nodeLinks[];
  nodes: routineCreate_routineCreate_nodes[];
  outputs: routineCreate_routineCreate_outputs[];
  owner: routineCreate_routineCreate_owner | null;
  parent: routineCreate_routineCreate_parent | null;
  reportsCount: number;
  resourceLists: routineCreate_routineCreate_resourceLists[];
  runs: routineCreate_routineCreate_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: routineCreate_routineCreate_permissionsRoutine;
  tags: routineCreate_routineCreate_tags[];
  translations: routineCreate_routineCreate_translations[];
  updated_at: any;
}

export interface routineCreate {
  routineCreate: routineCreate_routineCreate;
}

export interface routineCreateVariables {
  input: RoutineCreateInput;
}
