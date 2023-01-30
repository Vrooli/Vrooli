import { CopyResult } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const copyResultPartial: GqlPartial<CopyResult> = {
    __typename: 'CopyResult',
    common: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./noteVersion').noteVersionPartial, 'list'),
            2: () => relPartial(require('./organization').organizationPartial, 'list'),
            3: () => relPartial(require('./projectVersion').projectVersionPartial, 'list'),
            4: () => relPartial(require('./routineVersion').routineVersionPartial, 'list'),
            5: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'list'),
            6: () => relPartial(require('./standardVersion').standardVersionPartial, 'list'),
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