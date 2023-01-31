import { View } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const viewPartial: GqlPartial<View> = {
    __typename: 'View',
    list: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./issue')).issuePartial, 'list'),
            2: async () => relPartial((await import('./note')).notePartial, 'list'),
            3: async () => relPartial((await import('./organization')).organizationPartial, 'list'),
            4: async () => relPartial((await import('./post')).postPartial, 'list'),
            5: async () => relPartial((await import('./project')).projectPartial, 'list'),
            6: async () => relPartial((await import('./question')).questionPartial, 'list'),
            7: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            8: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            9: async () => relPartial((await import('./standard')).standardPartial, 'list'),
            10: async () => relPartial((await import('./user')).userPartial, 'list'),
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Issue: 1,
                Note: 2,
                Organization: 3,
                Post: 4,
                Project: 5,
                Question: 6,
                Routine: 7,
                SmartContract: 8,
                Standard: 9,
                User: 10
            }
        }
    }
}