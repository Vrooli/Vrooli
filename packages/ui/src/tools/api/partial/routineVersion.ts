import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const routineVersionTranslationPartial: GqlPartial<RoutineVersionTranslation> = {
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

export const routineVersionYouPartial: GqlPartial<RoutineVersionYou> = {
    __typename: 'RoutineVersionYou',
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canRun: true,
        canView: true,
        canVote: true,
    },
    full: {
        runs: async () => relPartial((await import('./runRoutine')).runRoutinePartial, 'full', { omit: 'routineVersion' }),
    },
    list: {
        runs: async () => relPartial((await import('./runRoutine')).runRoutinePartial, 'full', { omit: 'routineVersion' }),
    }
}

export const routineVersionPartial: GqlPartial<RoutineVersion> = {
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
        you: () => relPartial(routineVersionYouPartial, 'full'),
    },
    full: {
        versionNotes: true,
        apiVersion: async () => relPartial((await import('./apiVersion')).apiVersion, 'full'),
        inputs: async () => relPartial((await import('./routineVersionInput')).routineVersionInputPartial, 'full'),
        nodes: async () => relPartial((await import('./node')).nodePartial, 'full'),
        nodeLinks: async () => relPartial((await import('./nodeLink')).nodeLinkPartial, 'full'),
        outputs: async () => relPartial((await import('./routineVersionOutput')).routineVersionOutputPartial, 'full'),
        pullRequest: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'full'),
        resourceList: async () => relPartial((await import('./resourceList')).resourceListPartial, 'full'),
        root: async () => relPartial((await import('./routine')).routinePartial, 'full', { omit: 'versions' }),
        smartContractVersion: async () => relPartial((await import('./smartContractVersion')).smartContractVersionPartial, 'full'),
        suggestedNextByRoutineVersion: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav'),
        translations: () => relPartial(routineVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(routineVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        root: async () => relPartial((await import('./routine')).routinePartial, 'nav'),
        translations: () => relPartial(routineVersionTranslationPartial, 'list'),
        versionIndex: true,
        versionLabel: true,
    }
}