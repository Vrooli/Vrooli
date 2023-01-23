import { Transfer, TransferYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const transferYouPartial: GqlPartial<TransferYou> = {
    __typename: 'TransferYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const transferPartial: GqlPartial<Transfer> = {
    __typename: 'Transfer',
    common: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./note').notePartial, 'list'],
            2: [require('./project').projectPartial, 'list'],
            3: [require('./routine').routinePartial, 'list'],
            4: [require('./smartContract').smartContractPartial, 'list'],
            5: [require('./standard').standardPartial, 'list'],
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
}