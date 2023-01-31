import { PullRequest, PullRequestYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const pullRequestYouPartial: GqlPartial<PullRequestYou> = {
    __typename: 'PullRequestYou',
    common: {
        canComment: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
    },
    full: {},
    list: {},
}

export const pullRequestPartial: GqlPartial<PullRequest> = {
    __typename: 'PullRequest',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./apiVersion')).apiVersion, 'list'),
            2: async () => relPartial((await import('./note')).notePartial, 'list'),
            3: async () => relPartial((await import('./noteVersion')).noteVersionPartial, 'list'),
            4: async () => relPartial((await import('./project')).projectPartial, 'list'),
            5: async () => relPartial((await import('./projectVersion')).projectVersionPartial, 'list'),
            6: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            7: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'list'),
            8: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            9: async () => relPartial((await import('./smartContractVersion')).smartContractVersionPartial, 'list'),
            10: async () => relPartial((await import('./standard')).standardPartial, 'list'),
            11: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'list'),
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
        createdBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
        you: () => relPartial(pullRequestYouPartial, 'full'),
    },
    full: {},
    list: {},
    nav: {
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        status: true,
    }
}