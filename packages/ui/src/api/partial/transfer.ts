import { Transfer, TransferYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./note').notePartial, 'list'),
            2: () => relPartial(require('./project').projectPartial, 'list'),
            3: () => relPartial(require('./routine').routinePartial, 'list'),
            4: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            5: () => relPartial(require('./standard').standardPartial, 'list'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: () => relPartial(require('./user').userPartial, 'nav'),
        toOwner: () => relPartial(require('./user').userPartial, 'nav'),
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