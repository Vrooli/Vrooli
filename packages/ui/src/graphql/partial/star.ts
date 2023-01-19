import { Star } from "@shared/consts";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { commentPartial } from "./comment";
import { issuePartial } from "./issue";
import { notePartial } from "./note";
import { organizationPartial } from "./organization";
import { postPartial } from "./post";
import { projectPartial } from "./project";
import { questionPartial } from "./question";
import { questionAnswerPartial } from "./questionAnswer";
import { quizPartial } from "./quiz";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

export const starPartial: GqlPartial<Star> = {
    __typename: 'Star',
    list: {
        __define: {
            0: [apiPartial, 'list'],
            1: [commentPartial, 'list'],
            2: [issuePartial, 'list'],
            3: [notePartial, 'list'],
            4: [organizationPartial, 'list'],
            5: [postPartial, 'list'],
            6: [projectPartial, 'list'],
            7: [questionPartial, 'list'],
            8: [questionAnswerPartial, 'list'],
            9: [quizPartial, 'list'],
            10: [routinePartial, 'list'],
            11: [smartContractPartial, 'list'],
            12: [standardPartial, 'list'],
            13: [tagPartial, 'list'],
            14: [userPartial, 'list']
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Comment: 1,
                Issue: 2,
                Note: 3,
                Organization: 4,
                Post: 5,
                Project: 6,
                Question: 7,
                QuestionAnswer: 8,
                Quiz: 9,
                Routine: 10,
                SmartContract: 11,
                Standard: 12,
                Tag: 13,
                User: 14,
            }
        }
    }
}