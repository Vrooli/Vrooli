import { toFragment } from "graphql/utils";
import { listApiFields } from "./api";
import { listIssueFields } from "./issue";
import { listNoteFields } from "./note";
import { listOrganizationFields } from "./organization";
import { listPostFields } from "./post";
import { listProjectFields } from "./project";
import { listQuestionFields } from "./question";
import { listRoutineFields } from "./routine";
import { listSmartContractFields } from "./smartContract";
import { listStandardFields } from "./standard";
import { listUserFields } from "./user";

const __typename = 'View';
export const listViewFields = [__typename, `
    ${toFragment('listViewApiFields', listApiFields)}
    ${toFragment('listViewIssueFields', listIssueFields)}
    ${toFragment('listViewNoteFields', listNoteFields)}
    ${toFragment('listViewOrganizationFields', listOrganizationFields)}
    ${toFragment('listViewPostFields', listPostFields)}
    ${toFragment('listViewProjectFields', listProjectFields)}
    ${toFragment('listViewQuestionFields', listQuestionFields)}
    ${toFragment('listViewRoutineFields', listRoutineFields)}
    ${toFragment('listViewSmartContractFields', listSmartContractFields)}
    ${toFragment('listViewStandardFields', listStandardFields)}
    ${toFragment('listViewUserFields', listUserFields)}
    fragment listViewFields on View {
        id
        to {
            ... on Api { ...listViewApiFields }
            ... on Issue { ...listViewIssueFields }
            ... on Note { ...listViewNoteFields }
            ... on Organization { ...listViewOrganizationFields }
            ... on Post { ...listViewPostFields }
            ... on Project { ...listViewProjectFields }
            ... on Question { ...listViewQuestionFields }
            ... on Routine { ...listViewRoutineFields }
            ... on SmartContract { ...listViewSmartContractFields }
            ... on Standard { ...listViewStandardFields }
            ... on User { ...listViewUserFields }
        }
    }
`] as const;