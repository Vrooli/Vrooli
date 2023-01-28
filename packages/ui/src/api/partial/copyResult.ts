import { CopyResult } from "@shared/consts";
import { GqlPartial } from "types";

export const copyResultPartial: GqlPartial<CopyResult> = {
    __typename: 'CopyResult',
    common: {
        __define: {
            0: [require('./apiVersion').apiVersionPartial, 'list'],
            1: [require('./noteVersion').noteVersionPartial, 'list'],
            2: [require('./organization').organizationPartial, 'list'],
            3: [require('./projectVersion').projectVersionPartial, 'list'],
            4: [require('./routineVersion').routineVersionPartial, 'list'],
            5: [require('./smartContractVersion').smartContractVersionPartial, 'list'],
            6: [require('./standardVersion').standardVersionPartial, 'list'],
        },
        apiVersion: { __use: 0 },
        noteVersion: { __use: 1 },
        organization: { __use: 2 },
        projectVersion: { __use: 3 },
        routineVersion: { __use: 4 },
        smartContractVersion: { __use: 5 },
        standardVersion: { __use: 6 }
    },
    full: {},
    list: {},
}