import { Vote } from "@shared/consts";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { commentPartial } from "./comment";
import { issuePartial } from "./issue";
import { notePartial } from "./note";
import { postPartial } from "./post";
import { projectPartial } from "./project";
import { questionPartial } from "./question";
import { questionAnswerPartial } from "./questionAnswer";
import { quizPartial } from "./quiz";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";

export const votePartial: GqlPartial<Vote> = {
    __typename: 'Vote',
    list: {
        __define: {
            0: [apiPartial, 'list'],
            1: [commentPartial, 'list'],
            2: [issuePartial, 'list'],
            3: [notePartial, 'list'],
            4: [postPartial, 'list'],
            5: [projectPartial, 'list'],
            6: [questionPartial, 'list'],
            7: [questionAnswerPartial, 'list'],
            8: [quizPartial, 'list'],
            9: [routinePartial, 'list'],
            10: [smartContractPartial, 'list'],
            11: [standardPartial, 'list'],
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Comment: 1,
                Issue: 2,
                Note: 3,
                Post: 4,
                Project: 5,
                Question: 6,
                QuestionAnswer: 7,
                Quiz: 8,
                Routine: 9,
                SmartContract: 10,
                Standard: 11
            }
        }
    }
}