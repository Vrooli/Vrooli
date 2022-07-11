/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, MemberRole, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: routineNodeFields
// ====================================================

export interface routineNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_inputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_inputs_standard_translations[];
  version: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_inputs_translations[];
  standard: routineNodeFields_data_NodeRoutineList_routines_routine_inputs_standard | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_outputs_standard {
  __typename: "Standard";
  id: string;
  default: string | null;
  isInternal: boolean;
  name: string;
  type: string;
  props: string;
  yup: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_outputs_standard_translations[];
  version: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_outputs_translations[];
  standard: routineNodeFields_data_NodeRoutineList_routines_routine_outputs_standard | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization_translations[];
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type routineNodeFields_data_NodeRoutineList_routines_routine_owner = routineNodeFields_data_NodeRoutineList_routines_routine_owner_Organization | routineNodeFields_data_NodeRoutineList_routines_routine_owner_User;

export interface routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_resources_translations[];
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_translations[];
  resources: routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists_resources[];
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_tags {
  __typename: "Tag";
  tag: string;
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_tags_translations[];
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  title: string;
  description: string | null;
  instructions: string;
}

export interface routineNodeFields_data_NodeRoutineList_routines_routine {
  __typename: "Routine";
  id: string;
  complexity: number;
  isComplete: boolean;
  isInternal: boolean | null;
  inputs: routineNodeFields_data_NodeRoutineList_routines_routine_inputs[];
  nodesCount: number | null;
  role: MemberRole | null;
  outputs: routineNodeFields_data_NodeRoutineList_routines_routine_outputs[];
  owner: routineNodeFields_data_NodeRoutineList_routines_routine_owner | null;
  resourceLists: routineNodeFields_data_NodeRoutineList_routines_routine_resourceLists[];
  simplicity: number;
  tags: routineNodeFields_data_NodeRoutineList_routines_routine_tags[];
  translations: routineNodeFields_data_NodeRoutineList_routines_routine_translations[];
  version: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string | null;
}

export interface routineNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routine: routineNodeFields_data_NodeRoutineList_routines_routine;
  translations: routineNodeFields_data_NodeRoutineList_routines_translations[];
}

export interface routineNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: routineNodeFields_data_NodeRoutineList_routines[];
}

export type routineNodeFields_data = routineNodeFields_data_NodeEnd | routineNodeFields_data_NodeRoutineList;

export interface routineNodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineNodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: routineNodeFields_loop_whiles_translations[];
}

export interface routineNodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: routineNodeFields_loop_whiles[];
}

export interface routineNodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface routineNodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: routineNodeFields_data | null;
  loop: routineNodeFields_loop | null;
  translations: routineNodeFields_translations[];
}
