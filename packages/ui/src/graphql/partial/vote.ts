import { toFragment } from "graphql/utils";
import { listApiFields } from "./api";
import { commentFields } from "./comment";
import { listIssueFields } from "./issue";
import { listNoteFields } from "./note";
import { listPostFields } from "./post";
import { listProjectFields } from "./project";
import { listQuestionFields } from "./question";
import { listQuestionAnswerFields } from "./questionAnswer";
import { listQuizFields } from "./quiz";
import { listRoutineFields } from "./routine";
import { listSmartContractFields } from "./smartContract";
import { listStandardFields } from "./standard";

const __typename = 'Vote';
export const listVoteFields = [type, `
    ${toFragment('listVoteApiFields', listApiFields)}
    ${toFragment('listVoteCommentFields', commentFields)}
    ${toFragment('listVoteIssueFields', listIssueFields)}
    ${toFragment('listVoteNoteFields', listNoteFields)}
    ${toFragment('listVotePostFields', listPostFields)}
    ${toFragment('listVoteProjectFields', listProjectFields)}
    ${toFragment('listVoteQuestionFields', listQuestionFields)}
    ${toFragment('listVoteQuestionAnswerFields', listQuestionAnswerFields)}
    ${toFragment('listVoteQuizFields', listQuizFields)}
    ${toFragment('listVoteRoutineFields', listRoutineFields)}
    ${toFragment('listVoteSmartContractFields', listSmartContractFields)}
    ${toFragment('listVoteStandardFields', listStandardFields)}
    fragment listVoteFields on Vote {
        id
        to {
            ... on Api { ...listVoteApiFields }
            ... on Comment { ...listVoteCommentFields }
            ... on Issue { ...listVoteIssueFields }
            ... on Note { ...listVoteNoteFields }
            ... on Post { ...listVotePostFields }
            ... on Project { ...listVoteProjectFields }
            ... on Question { ...listVoteQuestionFields }
            ... on QuestionAnswer { ...listVoteQuestionAnswerFields }
            ... on Quiz { ...listVoteQuizFields }
            ... on Routine { ...listVoteRoutineFields }
            ... on SmartContract { ...listVoteSmartContractFields }
            ... on Standard { ...listVoteStandardFields }
        }
    }
`] as const;