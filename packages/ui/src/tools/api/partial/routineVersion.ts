import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

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
        canStar: true,
        canReport: true,
        canRun: true,
        canUpdate: true,
        canRead: true,
        canVote: true,
    },
    full: {
        runs: async () => rel((await import('./runRoutine')).runRoutine, 'full', { omit: 'routineVersion' }),
    },
    list: {
        runs: async () => rel((await import('./runRoutine')).runRoutine, 'full', { omit: 'routineVersion' }),
    }
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
        versionNotes: true,
        apiVersion: async () => rel((await import('./apiVersion')).apiVersion, 'full'),
        inputs: async () => rel((await import('./routineVersionInput')).routineVersionInput, 'full'),
        nodes: async () => rel((await import('./node')).node, 'full'),
        nodeLinks: async () => rel((await import('./nodeLink')).nodeLink, 'full'),
        outputs: async () => rel((await import('./routineVersionOutput')).routineVersionOutput, 'full'),
        pullRequest: async () => rel((await import('./pullRequest')).pullRequest, 'full', { omit: ['from', 'to'] }),
        resourceList: async () => rel((await import('./resourceList')).resourceList, 'full'),
        root: async () => rel((await import('./routine')).routine, 'full', { omit: 'versions' }),
        smartContractVersion: async () => rel((await import('./smartContractVersion')).smartContractVersion, 'full'),
        suggestedNextByRoutineVersion: async () => rel((await import('./routineVersion')).routineVersion, 'nav'),
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