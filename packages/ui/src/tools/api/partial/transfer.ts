import { Transfer, TransferYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const transferYouPartial: GqlPartial<TransferYou> = {
    __typename: 'TransferYou',
    common: {
        canDelete: true,
        canEdit: true,
    },
    full: {},
    list: {},
}

export const transferPartial: GqlPartial<Transfer> = {
    __typename: 'Transfer',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./note')).notePartial, 'list'),
            2: async () => relPartial((await import('./project')).projectPartial, 'list'),
            3: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            4: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            5: async () => relPartial((await import('./standard')).standardPartial, 'list'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: async () => relPartial((await import('./user')).userPartial, 'nav'),
        toOwner: async () => relPartial((await import('./user')).userPartial, 'nav'),
        object: {
            __union: {
                Api: 0,
                Note: 1,
                Project: 2,
                Routine: 3,
                SmartContract: 4,
                Standard: 5,
            }
        },
        you: () => relPartial(transferYouPartial, 'full'),
    },
    full: {},
    list: {},
}