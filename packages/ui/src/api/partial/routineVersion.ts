import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
        runs: () => relPartial(require('./runRoutine').runRoutinePartial, 'full', { omit: 'routineVersion' }),
    },
    list: {
        runs: () => relPartial(require('./runRoutine').runRoutinePartial, 'full', { omit: 'routineVersion' }),
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
        apiVersion: () => relPartial(require('./apiVersion').apiVersionPartial, 'full'),
        inputs: () => relPartial(require('./routineVersionInput').routineVersionInputPartial, 'full'),
        nodes: () => relPartial(require('./node').nodePartial, 'full'),
        nodeLinks: () => relPartial(require('./nodeLink').nodeLinkPartial, 'full'),
        outputs: () => relPartial(require('./routineVersionOutput').routineVersionOutputPartial, 'full'),
        pullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'full'),
        resourceList: () => relPartial(require('./resourceList').resourceListPartial, 'full'),
        root: () => relPartial(require('./routine').routinePartial, 'full', { omit: 'versions' }),
        smartContractVersion: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'full'),
        suggestedNextByRoutineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
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
        translations: () => relPartial(routineVersionTranslationPartial, 'full'),
        versionIndex: true,
        versionLabel: true,
    }
}