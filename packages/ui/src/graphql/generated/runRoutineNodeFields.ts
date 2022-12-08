/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NodeType, ResourceListUsedFor, ResourceUsedFor } from "./globalTypes";

// ====================================================
// GraphQL fragment: runRoutineNodeFields
// ====================================================

export interface runRoutineNodeFields_data_NodeEnd {
  __typename: "NodeEnd";
  id: string;
  wasSuccessful: boolean;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_translations {
  __typename: "InputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard {
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
  tags: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_tags[];
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs {
  __typename: "InputItem";
  id: string;
  isRequired: boolean | null;
  name: string | null;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_translations[];
  standard: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs_standard | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_translations {
  __typename: "OutputItemTranslation";
  id: string;
  language: string;
  description: string | null;
  helpText: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard {
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
  tags: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_tags[];
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs {
  __typename: "OutputItem";
  id: string;
  name: string | null;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_translations[];
  standard: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs_standard | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_Organization_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_User {
  __typename: "User";
  id: string;
  name: string;
  handle: string | null;
}

export type runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner = runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_Organization | runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner_User;

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_permissionsRoutine {
  __typename: "RoutinePermission";
  canDelete: boolean;
  canEdit: boolean;
  canFork: boolean;
  canStar: boolean;
  canReport: boolean;
  canRun: boolean;
  canVote: boolean;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_translations {
  __typename: "ResourceListTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations {
  __typename: "ResourceTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_resources {
  __typename: "Resource";
  id: string;
  created_at: any;
  index: number | null;
  link: string;
  updated_at: any;
  usedFor: ResourceUsedFor | null;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_resources_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists {
  __typename: "ResourceList";
  id: string;
  created_at: any;
  index: number | null;
  usedFor: ResourceListUsedFor | null;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_translations[];
  resources: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists_resources[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_tags {
  __typename: "Tag";
  tag: string;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_tags_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
  instructions: string;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion {
  __typename: "Routine";
  id: string;
  complexity: number;
  inputs: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_inputs[];
  isComplete: boolean;
  isDeleted: boolean;
  isInternal: boolean | null;
  isPrivate: boolean;
  nodesCount: number | null;
  outputs: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_outputs[];
  owner: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_owner | null;
  permissionsRoutine: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_permissionsRoutine;
  resourceLists: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_resourceLists[];
  simplicity: number;
  tags: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_tags[];
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines_translations {
  __typename: "NodeRoutineListItemTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string | null;
}

export interface runRoutineNodeFields_data_NodeRoutineList_routines {
  __typename: "NodeRoutineListItem";
  id: string;
  index: number;
  isOptional: boolean;
  routineVersion: runRoutineNodeFields_data_NodeRoutineList_routines_routineVersion;
  translations: runRoutineNodeFields_data_NodeRoutineList_routines_translations[];
}

export interface runRoutineNodeFields_data_NodeRoutineList {
  __typename: "NodeRoutineList";
  id: string;
  isOptional: boolean;
  isOrdered: boolean;
  routines: runRoutineNodeFields_data_NodeRoutineList_routines[];
}

export type runRoutineNodeFields_data = runRoutineNodeFields_data_NodeEnd | runRoutineNodeFields_data_NodeRoutineList;

export interface runRoutineNodeFields_loop_whiles_translations {
  __typename: "LoopWhileTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineNodeFields_loop_whiles {
  __typename: "LoopWhile";
  id: string;
  condition: string;
  translations: runRoutineNodeFields_loop_whiles_translations[];
}

export interface runRoutineNodeFields_loop {
  __typename: "Loop";
  id: string;
  loops: number | null;
  maxLoops: number | null;
  operation: string | null;
  whiles: runRoutineNodeFields_loop_whiles[];
}

export interface runRoutineNodeFields_translations {
  __typename: "NodeTranslation";
  id: string;
  language: string;
  description: string | null;
  name: string;
}

export interface runRoutineNodeFields {
  __typename: "Node";
  id: string;
  columnIndex: number | null;
  created_at: any;
  rowIndex: number | null;
  type: NodeType;
  updated_at: any;
  data: runRoutineNodeFields_data | null;
  loop: runRoutineNodeFields_loop | null;
  translations: runRoutineNodeFields_translations[];
}
