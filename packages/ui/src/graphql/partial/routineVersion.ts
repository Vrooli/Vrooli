import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiVersionPartial } from "./apiVersion";
import { pullRequestPartial } from "./pullRequest";
import { resourceListPartial } from "./resourceList";
import { routinePartial } from "./routine";
import { runRoutinePartial } from "./runRoutine";
import { smartContractVersionPartial } from "./smartContractVersion";

export const routineVersionTranslationPartial: GqlPartial<RoutineVersionTranslation> = {
    __typename: 'RoutineVersionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        instructions: true,
        name: true,
    },
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
        runs: () => relPartial(runRoutinePartial, 'full', { omit: 'routineVersion' }),
    },
    list: {
        runs: () => relPartial(runRoutinePartial, 'full', { omit: 'routineVersion' }),
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
        outputsCount: true,
        reportsCount: true,
        you: () => relPartial(routineVersionYouPartial, 'full'),
    },
    full: {
        versionNotes: true,
        apiVersion: () => relPartial(apiVersionPartial, 'full'),
        inputs: () => relPartial(routineVersionInputPartial, 'full'),
        nodes: () => relPartial(nodePartial, 'full'),
        outputs: () => relPartial(routineVersionOutputPartial, 'full'),
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        resourceList: () => relPartial(resourceListPartial, 'full'),
        root: () => relPartial(routinePartial, 'full', { omit: 'versions' }),
        smartContractVersion: () => relPartial(smartContractVersionPartial, 'full'),
        suggestedNextByRoutineVersion: () => relPartial(routineVersionPartial, 'nav'),
        translations: () => relPartial(routineVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(routineVersionTranslationPartial, 'list'),
    }
}