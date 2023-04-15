import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const routineVersionTranslation: GqlPartial<RoutineVersionTranslation> = {
    __typename: 'RoutineVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        instructions: true,
        name: true,
    },
    full: {},
    list: {},
}

export const routineVersionYou: GqlPartial<RoutineVersionYou> = {
    __typename: 'RoutineVersionYou',
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canBookmark: true,
        canReport: true,
        canRun: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
    },
    full: {
        runs: async () => rel((await import('./runRoutine')).runRoutine, 'full', { omit: 'routineVersion' }),
    },
}

export const routineVersion: GqlPartial<RoutineVersion> = {
    __typename: 'RoutineVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        completedAt: true,
        complexity: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        simplicity: true,
        timesStarted: true,
        timesCompleted: true,
        smartContractCallData: true,
        apiCallData: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        nodesCount: true,
        nodeLinksCount: true,
        outputsCount: true,
        reportsCount: true,
        you: () => rel(routineVersionYou, 'full'),
    },
    full: {
        __define: {
            0: async () => rel((await import('./apiVersion')).apiVersion, 'full'),
            1: async () => rel((await import('./routineVersionInput')).routineVersionInput, 'full'),
            2: async () => rel((await import('./node')).node, 'full', { omit: 'routineVersion' }),
            3: async () => rel((await import('./nodeLink')).nodeLink, 'full'),
            4: async () => rel((await import('./routineVersionOutput')).routineVersionOutput, 'full'),
            5: async () => rel((await import('./pullRequest')).pullRequest, 'full', { omit: ['from', 'to'] }),
            6: async () => rel((await import('./resourceList')).resourceList, 'full'),
            7: async () => rel((await import('./routine')).routine, 'full', { omit: 'versions' }),
            8: async () => rel((await import('./smartContractVersion')).smartContractVersion, 'full'),
            9: async () => rel((await import('./routineVersion')).routineVersion, 'nav'),
        },
        versionNotes: true,
        apiVersion: { __use: 0 },
        inputs: { __use: 1 },
        nodes: { __use: 2 },
        nodeLinks: { __use: 3 },
        outputs: { __use: 4 },
        pullRequest: { __use: 5 },
        resourceList: { __use: 6 },
        root: { __use: 7 },
        smartContractVersion: { __use: 8 },
        suggestedNextByRoutineVersion: { __use: 9 },
        translations: () => rel(routineVersionTranslation, 'full'),
    },
    list: {
        translations: () => rel(routineVersionTranslation, 'list'),
    },
    nav: {
        id: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        root: async () => rel((await import('./routine')).routine, 'nav'),
        translations: () => rel(routineVersionTranslation, 'list'),
        versionIndex: true,
        versionLabel: true,
    }
}