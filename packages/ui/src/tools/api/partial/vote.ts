import { Vote } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const vote: GqlPartial<Vote> = {
    __typename: 'Vote',
    list: {
        __define: {
            0: async () => rel((await import('./api')).api, 'list'),
            1: async () => rel((await import('./comment')).comment, 'list'),
            2: async () => rel((await import('./issue')).issue, 'list'),
            3: async () => rel((await import('./note')).note, 'list'),
            4: async () => rel((await import('./post')).post, 'list'),
            5: async () => rel((await import('./project')).project, 'list'),
            6: async () => rel((await import('./question')).question, 'list'),
            7: async () => rel((await import('./questionAnswer')).questionAnswer, 'list'),
            8: async () => rel((await import('./quiz')).quiz, 'list'),
            9: async () => rel((await import('./routine')).routine, 'list'),
            10: async () => rel((await import('./smartContract')).smartContract, 'list'),
            11: async () => rel((await import('./standard')).standard, 'list'),
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