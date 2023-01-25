import { Star } from "@shared/consts";
import { GqlPartial } from "types";

export const starPartial: GqlPartial<Star> = {
    __typename: 'Star',
    list: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./comment').commentPartial, 'list'],
            2: [require('./issue').issuePartial, 'list'],
            3: [require('./note').notePartial, 'list'],
            4: [require('./organization').organizationPartial, 'list'],
            5: [require('./post').postPartial, 'list'],
            6: [require('./project').projectPartial, 'list'],
            7: [require('./question').questionPartial, 'list'],
            8: [require('./questionAnswer').questionAnswerPartial, 'list'],
            9: [require('./quiz').quizPartial, 'list'],
            10: [require('./routine').routinePartial, 'list'],
            11: [require('./smartContract').smartContractPartial, 'list'],
            12: [require('./standard').standardPartial, 'list'],
            13: [require('./tag').tagPartial, 'list'],
            14: [require('./user').userPartial, 'list']
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