import { Transfer, TransferYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { notePartial } from "./note";
import { projectPartial } from "./project";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";
import { userPartial } from "./user";

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
            0: [apiPartial, 'list'],
            1: [notePartial, 'list'],
            2: [projectPartial, 'list'],
            3: [routinePartial, 'list'],
            4: [smartContractPartial, 'list'],
            5: [standardPartial, 'list'],
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
        fromOwner: () => relPartial(userPartial, 'nav'),
        toOwner: () => relPartial(userPartial, 'nav'),
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