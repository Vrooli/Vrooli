/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, ResourceListUsedFor, ResourceUsedFor, RunStatus, RunStepStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineFields
// ====================================================

export interface routineFields_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineFields_inputs_standard {
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
  translations: routineFields_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routineFields_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineFields_inputs_translations[];
  standard: routineFields_inputs_standard | null;
}

export interface routineFields_nodeLinks_whens_translations {
  __typename: "NodeLinkWhenTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineFields_nodeLinks_whens {
  __typename: "NodeLinkWhen";
  id: string;
  condition: string;
  translations: routineFields_nodeLinks_whens_translations[];
}

export interface routineFields_nodeLinks {
  __typename: "NodeLink";
  id: string;
  fromId: string;
  toId: string;
  operation: string | null;
  whens: routineFields_nodeLinks_whens[];
}

export interface routineFields_nodes_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_standard {
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
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: routineFields_nodes_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_standard {
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
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: routineFields_nodes_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineFields_nodes_data_NodeRoutineList_routines_routine_owner = routineFields_nodes_data_NodeRoutineList_routines_routine_owner_Organization | routineFields_nodes_data_NodeRoutineList_routines_routine_owner_User;

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  inputs: routineFields_nodes_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  outputs: routineFields_nodes_data_NodeRoutineList_routines_routine_outputs[];
  owner: routineFields_nodes_data_NodeRoutineList_routines_routine_owner | null;
  permissionsRoutine: routineFields_nodes_data_NodeRoutineList_routines_routine_permissionsRoutine;
  resourceLists: routineFields_nodes_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: routineFields_nodes_data_NodeRoutineList_routines_routine_tags[];
  translations: routineFields_nodes_data_NodeRoutineList_routines_routine_translations[];
  version: string;
  versionGroupId: string;
}

export interface routineFields_nodes_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineFields_nodes_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routineFields_nodes_data_NodeRoutineList_routines_routine;
  translations: routineFields_nodes_data_NodeRoutineList_routines_translations[];
}

export interface routineFields_nodes_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineFields_nodes_data_NodeRoutineList_routines[];
}

export type routineFields_nodes_data = routineFields_nodes_data_NodeEnd | routineFields_nodes_data_NodeRoutineList;

export interface routineFields_nodes_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineFields_nodes_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineFields_nodes_loop_whiles_translations[];
}

export interface routineFields_nodes_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineFields_nodes_loop_whiles[];
}

export interface routineFields_nodes_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineFields_nodes {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineFields_nodes_data | null;
  loop: routineFields_nodes_loop | null;
  translations: routineFields_nodes_translations[];
}

export interface routineFields_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineFields_outputs_standard {
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
  translations: routineFields_outputs_standard_translations[];
  version: string;
  versionGroupId: string;
}

export interface routineFields_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineFields_outputs_translations[];
  standard: routineFields_outputs_standard | null;
}

export interface routineFields_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineFields_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineFields_owner_Organization_translations[];
}

export interface routineFields_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineFields_owner = routineFields_owner_Organization | routineFields_owner_User;

export interface routineFields_parent_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
}

export interface routineFields_parent {
  __typename: "Routine";
  id: string;
  translations: routineFields_parent_translations[];
}

export interface routineFields_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineFields_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineFields_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineFields_resourceLists_resources_translations[];
}

export interface routineFields_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineFields_resourceLists_translations[];
  resources: routineFields_resourceLists_resources[];
}

export interface routineFields_runs_steps_node {
  __typename: "Node";
  id: string;
}

export interface routineFields_runs_steps {
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
  node: routineFields_runs_steps_node | null;
}

export interface routineFields_runs {
  __typename: "Run";
  id: string;
  completedComplexity: number;
  contextSwitches: number;
  timeStarted: any | null;
  timeElapsed: number | null;
  timeCompleted: any | null;
  title: string;
  status: RunStatus;
  steps: routineFields_runs_steps[];
}

export interface routineFields_permissionsRoutine {
  __typename: "RoutinePermission";
  canComment: boolean;
  canDelete: boolean;
  canEdit: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface routineFields_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineFields_tags {
  __typename: "Tag";
  tag: string;
  translations: routineFields_tags_translations[];
}

export interface routineFields_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  instructions: string;
  title: string;
}

export interface routineFields {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  inputs: routineFields_inputs[];
  isAutomatable: boolean | null;
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  nodeLinks: routineFields_nodeLinks[];
  nodes: routineFields_nodes[];
  outputs: routineFields_outputs[];
  owner: routineFields_owner | null;
  parent: routineFields_parent | null;
  resourceLists: routineFields_resourceLists[];
  runs: routineFields_runs[];
  score: number;
  simplicity: number;
  stars: number;
  permissionsRoutine: routineFields_permissionsRoutine;
  tags: routineFields_tags[];
  translations: routineFields_translations[];
  updated_at: any;
  version: string;
  versionGroupId: string;
}
