import { listOrganizationFields } from "./organization";
import { listProjectFields } from "./project";
import { listRoutineFields } from "./routine";

export const listProjectOrOrganizationFields = ['ProjectOrOrganization', `{
    ... on Project ${listProjectFields[1]}
    ... on Organization ${listOrganizationFields[1]}
}`] as const;

export const listProjectOrRoutineFields = ['ProjectOrRoutine', `{
    ... on Project ${listProjectFields[1]}
    ... on Routine ${listRoutineFields[1]}
}`] as const;