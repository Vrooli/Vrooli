import { Star } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const starPartial: GqlPartial<Star> = {
    __typename: 'Star',
    list: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./comment')).comment, 'list'),
            2: async () => relPartial((await import('./issue')).issuePartial, 'list'),
            3: async () => relPartial((await import('./note')).notePartial, 'list'),
            4: async () => relPartial((await import('./organization')).organizationPartial, 'list'),
            5: async () => relPartial((await import('./post')).postPartial, 'list'),
            6: async () => relPartial((await import('./project')).projectPartial, 'list'),
            7: async () => relPartial((await import('./question')).questionPartial, 'list'),
            8: async () => relPartial((await import('./questionAnswer')).questionAnswerPartial, 'list'),
            9: async () => relPartial((await import('./quiz')).quizPartial, 'list'),
            10: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            11: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            12: async () => relPartial((await import('./standard')).standardPartial, 'list'),
            13: async () => relPartial((await import('./tag')).tagPartial, 'list'),
            14: async () => relPartial((await import('./user')).userPartial, 'list'),
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