import { organizationPartial } from "./organization";

export const listProjectOrOrganizationFields = ['ProjectOrOrganization', `{
    ... on Project ${projectPartial.list}
    ... on Organization ${organizationPartial.list}
}`] as const;

export const listProjectOrRoutineFields = ['ProjectOrRoutine', `{
    ... on Project ${projectPartial.list}
    ... on Routine ${routinePartial.list}
}`] as const;

export const listRunProjectOrRunRoutineFields = ['RunProjectOrRunRoutine', `{
    ... on RunProject ${runProjectPartial.list}
    ... on RunRoutine ${runRoutinePartial.list}
}`] as const;