import { Vote } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const votePartial: GqlPartial<Vote> = {
    __typename: 'Vote',
    list: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./comment').commentPartial, 'list'),
            2: () => relPartial(require('./issue').issuePartial, 'list'),
            3: () => relPartial(require('./note').notePartial, 'list'),
            4: () => relPartial(require('./post').postPartial, 'list'),
            5: () => relPartial(require('./project').projectPartial, 'list'),
            6: () => relPartial(require('./question').questionPartial, 'list'),
            7: () => relPartial(require('./questionAnswer').questionAnswerPartial, 'list'),
            8: () => relPartial(require('./quiz').quizPartial, 'list'),
            9: () => relPartial(require('./routine').routinePartial, 'list'),
            10: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            11: () => relPartial(require('./standard').standardPartial, 'list'),
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