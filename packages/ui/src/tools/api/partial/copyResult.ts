import { CopyResult } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const copyResult: GqlPartial<CopyResult> = {
    __typename: 'CopyResult',
    common: {
        __define: {
            0: async () => rel((await import('./api')).api, 'list'),
            1: async () => rel((await import('./noteVersion')).noteVersion, 'list'),
            2: async () => rel((await import('./organization')).organization, 'list'),
            3: async () => rel((await import('./projectVersion')).projectVersion, 'list'),
            4: async () => rel((await import('./routineVersion')).routineVersion, 'list'),
            5: async () => rel((await import('./smartContractVersion')).smartContractVersion, 'list'),
            6: async () => rel((await import('./standardVersion')).standardVersion, 'list'),
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