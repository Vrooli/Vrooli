/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { MemberRole } from "./globalTypes";

// ====================================================
// GraphQL fragment: listViewFields
// ====================================================

export interface listViewFields_to_Organization_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewFields_to_Organization_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewFields_to_Organization_tags_translations[];
}

export interface listViewFields_to_Organization_translations {
  __typename: "OrganizationTranslation";
  id: string;
  language: string;
  name: string;
  bio: string | null;
}

export interface listViewFields_to_Organization {
  __typename: "Organization";
  id: string;
  handle: string | null;
  stars: number;
  isStarred: boolean;
  role: MemberRole | null;
  tags: listViewFields_to_Organization_tags[];
  translations: listViewFields_to_Organization_translations[];
}

export interface listViewFields_to_Project_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewFields_to_Project_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewFields_to_Project_tags_translations[];
}

export interface listViewFields_to_Project_translations {
  __typename: "ProjectTranslation";
  id: string;
  language: string;
  name: string;
  description: string | null;
}

export interface listViewFields_to_Project {
  __typename: "Project";
  id: string;
  handle: string | null;
  role: MemberRole | null;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  tags: listViewFields_to_Project_tags[];
  translations: listViewFields_to_Project_translations[];
}

export interface listViewFields_to_Routine_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewFields_to_Routine_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewFields_to_Routine_tags_translations[];
}

export interface listViewFields_to_Routine_translations {
  __typename: "RoutineTranslation";
  id: string;
  language: string;
  description: string | null;
  title: string;
}

export interface listViewFields_to_Routine {
  __typename: "Routine";
  id: string;
  completedAt: any | null;
  complexity: number;
  created_at: any;
  isAutomatable: boolean | null;
  isInternal: boolean | null;
  isComplete: boolean;
  isStarred: boolean;
  isUpvoted: boolean | null;
  role: MemberRole | null;
  score: number;
  simplicity: number;
  stars: number;
  tags: listViewFields_to_Routine_tags[];
  translations: listViewFields_to_Routine_translations[];
  version: string | null;
}

export interface listViewFields_to_Standard_tags_translations {
  __typename: "TagTranslation";
  id: string;
  language: string;
  description: string | null;
}

export interface listViewFields_to_Standard_tags {
  __typename: "Tag";
  created_at: any;
  isStarred: boolean;
  stars: number;
  tag: string;
  translations: listViewFields_to_Standard_tags_translations[];
}

export interface listViewFields_to_Standard_translations {
  __typename: "StandardTranslation";
  id: string;
  language: string;
  description: string | null;
  jsonVariable: string | null;
}

export interface listViewFields_to_Standard {
  __typename: "Standard";
  id: string;
  score: number;
  stars: number;
  isUpvoted: boolean | null;
  isStarred: boolean;
  name: string;
  role: MemberRole | null;
  tags: listViewFields_to_Standard_tags[];
  translations: listViewFields_to_Standard_translations[];
}

export interface listViewFields_to_User {
  __typename: "User";
  id: string;
  handle: string | null;
  name: string;
  stars: number;
  isStarred: boolean;
}

export type listViewFields_to = listViewFields_to_Organization | listViewFields_to_Project | listViewFields_to_Routine | listViewFields_to_Standard | listViewFields_to_User;

export interface listViewFields {
  __typename: "View";
  id: string;
  lastViewed: any;
  title: string;
  to: listViewFields_to;
}
