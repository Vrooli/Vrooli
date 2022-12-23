import { toFragment } from "graphql/utils";
import { listApiFields } from "./api";
import { commentFields } from "./comment";
import { listIssueFields } from "./issue";
import { listNoteFields } from "./note";
import { listOrganizationFields } from "./organization";
import { listPostFields } from "./post";
import { listProjectFields } from "./project";
import { listQuestionFields } from "./question";
import { listQuestionAnswerFields } from "./questionAnswer";
import { listQuizFields } from "./quiz";
import { listRoutineFields } from "./routine";
import { listSmartContractFields } from "./smartContract";
import { listStandardFields } from "./standard";
import { tagFields } from "./tag";
import { listUserFields } from "./user";

const __typename = 'Star';
export const listStarFields = [__typename, `
    ${toFragment('listStarTagFields', tagFields)}
    ${toFragment('listStarApiFields', listApiFields)}
    ${toFragment('listStarCommentFields', commentFields)}
    ${toFragment('listStarIssueFields', listIssueFields)}
    ${toFragment('listStarNoteFields', listNoteFields)}
    ${toFragment('listStarOrganizationFields', listOrganizationFields)}
    ${toFragment('listStarPostFields', listPostFields)}
    ${toFragment('listStarProjectFields', listProjectFields)}
    ${toFragment('listStarQuestionFields', listQuestionFields)}
    ${toFragment('listStarQuestionAnswerFields', listQuestionAnswerFields)}
    ${toFragment('listStarQuizFields', listQuizFields)}
    ${toFragment('listStarRoutineFields', listRoutineFields)}
    ${toFragment('listStarSmartContractFields', listSmartContractFields)}
    ${toFragment('listStarStandardFields', listStandardFields)}
    ${toFragment('listStarUserFields', listUserFields)}
    fragment listStarFields on Star {
        id
        to {
            ... on Api { ...listStarApiFields }
            ... on Comment { ...listStarCommentFields }
            ... on Issue { ...listStarIssueFields }
            ... on Note { ...listStarNoteFields }
            ... on Organization { ...listStarOrganizationFields }
            ... on Post { ...listStarPostFields }
            ... on Project { ...listStarProjectFields }
            ... on Question { ...listStarQuestionFields }
            ... on QuestionAnswer { ...listStarQuestionAnswerFields }
            ... on Quiz { ...listStarQuizFields }
            ... on Routine { ...listStarRoutineFields }
            ... on SmartContract { ...listStarSmartContractFields }
            ... on Standard { ...listStarStandardFields }
            ... on Tag { ...listStarTagFields }
            ... on User { ...listStarUserFields }
        }
    }
`] as const;