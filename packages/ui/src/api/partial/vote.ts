import { Vote } from "@shared/consts";
import { GqlPartial } from "types";

export const votePartial: GqlPartial<Vote> = {
    __typename: 'Vote',
    list: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./comment').commentPartial, 'list'],
            2: [require('./issue').issuePartial, 'list'],
            3: [require('./note').notePartial, 'list'],
            4: [require('./post').postPartial, 'list'],
            5: [require('./project').projectPartial, 'list'],
            6: [require('./question').questionPartial, 'list'],
            7: [require('./questionAnswer').questionAnswerPartial, 'list'],
            8: [require('./quiz').quizPartial, 'list'],
            9: [require('./routine').routinePartial, 'list'],
            10: [require('./smartContract').smartContractPartial, 'list'],
            11: [require('./standard').standardPartial, 'list'],
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