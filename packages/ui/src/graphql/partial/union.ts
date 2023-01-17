import { listOrganizationFields } from "./organization";
import { listProjectFields } from "./project";
import { listRoutineFields } from "./routine";
import { listRunProjectFields } from "./runProject";
import { listRunRoutineFields } from "./runRoutine";

export const listProjectOrOrganizationFields = ['ProjectOrOrganization', `{
    ... on Project ${listProjectFields[1]}
    ... on Organization ${listOrganizationFields[1]}
}`] as const;

export const listProjectOrRoutineFields = ['ProjectOrRoutine', `{
    ... on Project ${listProjectFields[1]}
    ... on Routine ${listRoutineFields[1]}
}`] as const;

export const listRunProjectOrRunRoutineFields = ['RunProjectOrRunRoutine', `{
    ... on RunProject ${listRunProjectFields[1]}
    ... on RunRoutine ${listRunRoutineFields[1]}
}`] as const;