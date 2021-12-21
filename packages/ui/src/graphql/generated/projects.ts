/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectsQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: projects
// ====================================================

export interface projects_projects_resources {
  __typename: "Resource";
  id: string;
  name: string;
  description: string | null;
  link: string;
  displayUrl: string | null;
}

export interface projects_projects {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  resources: projects_projects_resources[] | null;
}

export interface projects {
  projects: projects_projects[];
}

export interface projectsVariables {
  input: ProjectsQueryInput[];
}
