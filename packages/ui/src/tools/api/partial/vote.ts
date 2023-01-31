import { Vote } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const votePartial: GqlPartial<Vote> = {
    __typename: 'Vote',
    list: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./comment')).comment, 'list'),
            2: async () => relPartial((await import('./issue')).issuePartial, 'list'),
            3: async () => relPartial((await import('./note')).notePartial, 'list'),
            4: async () => relPartial((await import('./post')).postPartial, 'list'),
            5: async () => relPartial((await import('./project')).projectPartial, 'list'),
            6: async () => relPartial((await import('./question')).questionPartial, 'list'),
            7: async () => relPartial((await import('./questionAnswer')).questionAnswerPartial, 'list'),
            8: async () => relPartial((await import('./quiz')).quizPartial, 'list'),
            9: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            10: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            11: async () => relPartial((await import('./standard')).standardPartial, 'list'),
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