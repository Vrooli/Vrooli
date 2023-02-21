import { PullRequest, PullRequestTranslation, PullRequestYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const pullRequestYou: GqlPartial<PullRequestYou> = {
    __typename: 'PullRequestYou',
    common: {
        canComment: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
    },
    full: {},
    list: {},
}

export const pullRequestTranslation: GqlPartial<PullRequestTranslation> = {
    __typename: 'PullRequestTranslation',
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
}

export const pullRequest: GqlPartial<PullRequest> = {
    __typename: 'PullRequest',
    common: {
        __define: {
            0: async () => rel((await import('./api')).api, 'list'),
            1: async () => rel((await import('./apiVersion')).apiVersion, 'list'),
            2: async () => rel((await import('./note')).note, 'list'),
            3: async () => rel((await import('./noteVersion')).noteVersion, 'list'),
            4: async () => rel((await import('./project')).project, 'list'),
            5: async () => rel((await import('./projectVersion')).projectVersion, 'list'),
            6: async () => rel((await import('./routine')).routine, 'list'),
            7: async () => rel((await import('./routineVersion')).routineVersion, 'list'),
            8: async () => rel((await import('./smartContract')).smartContract, 'list'),
            9: async () => rel((await import('./smartContractVersion')).smartContractVersion, 'list'),
            10: async () => rel((await import('./standard')).standard, 'list'),
            11: async () => rel((await import('./standardVersion')).standardVersion, 'list'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        commentsCount: true,
        status: true,
        from: {
            __union: {
                ApiVersion: 1,
                NoteVersion: 3,
                ProjectVersion: 5,
                RoutineVersion: 7,
                SmartContractVersion: 9,
                StandardVersion: 11,
            }
        },
        to: {
            __union: {
                Api: 0,
                Note: 2,
                Project: 4,
                Routine: 6,
                SmartContract: 8,
                Standard: 10,
            }
        },
        createdBy: async () => rel((await import('./user')).user, 'nav'),
        you: () => rel(pullRequestYou, 'full'),
    },
    full: {
        translations: () => rel(pullRequestTranslation, 'full'),
    },
    list: {
        translations: () => rel(pullRequestTranslation, 'list'),
    },
    nav: {
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
    }
}