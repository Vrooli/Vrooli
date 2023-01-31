import { CopyResult } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const copyResultPartial: GqlPartial<CopyResult> = {
    __typename: 'CopyResult',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./noteVersion')).noteVersionPartial, 'list'),
            2: async () => relPartial((await import('./organization')).organizationPartial, 'list'),
            3: async () => relPartial((await import('./projectVersion')).projectVersionPartial, 'list'),
            4: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'list'),
            5: async () => relPartial((await import('./smartContractVersion')).smartContractVersionPartial, 'list'),
            6: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'list'),
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