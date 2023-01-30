import { Star } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const starPartial: GqlPartial<Star> = {
    __typename: 'Star',
    list: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./comment').commentPartial, 'list'),
            2: () => relPartial(require('./issue').issuePartial, 'list'),
            3: () => relPartial(require('./note').notePartial, 'list'),
            4: () => relPartial(require('./organization').organizationPartial, 'list'),
            5: () => relPartial(require('./post').postPartial, 'list'),
            6: () => relPartial(require('./project').projectPartial, 'list'),
            7: () => relPartial(require('./question').questionPartial, 'list'),
            8: () => relPartial(require('./questionAnswer').questionAnswerPartial, 'list'),
            9: () => relPartial(require('./quiz').quizPartial, 'list'),
            10: () => relPartial(require('./routine').routinePartial, 'list'),
            11: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            12: () => relPartial(require('./standard').standardPartial, 'list'),
            13: () => relPartial(require('./tag').tagPartial, 'list'),
            14: () => relPartial(require('./user').userPartial, 'list'),
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