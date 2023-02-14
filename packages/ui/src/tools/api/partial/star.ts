import { Bookmark } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const star: GqlPartial<Bookmark> = {
    __typename: 'Star',
    list: {
        __define: {
            0: async () => rel((await import('./api')).api, 'list'),
            1: async () => rel((await import('./comment')).comment, 'list'),
            2: async () => rel((await import('./issue')).issue, 'list'),
            3: async () => rel((await import('./note')).note, 'list'),
            4: async () => rel((await import('./organization')).organization, 'list'),
            5: async () => rel((await import('./post')).post, 'list'),
            6: async () => rel((await import('./project')).project, 'list'),
            7: async () => rel((await import('./question')).question, 'list'),
            8: async () => rel((await import('./questionAnswer')).questionAnswer, 'list'),
            9: async () => rel((await import('./quiz')).quiz, 'list'),
            10: async () => rel((await import('./routine')).routine, 'list'),
            11: async () => rel((await import('./smartContract')).smartContract, 'list'),
            12: async () => rel((await import('./standard')).standard, 'list'),
            13: async () => rel((await import('./tag')).tag, 'list'),
            14: async () => rel((await import('./user')).user, 'list'),
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