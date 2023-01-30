import { PullRequest, PullRequestYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./apiVersion').apiVersionPartial, 'list'),
            2: () => relPartial(require('./note').notePartial, 'list'),
            3: () => relPartial(require('./noteVersion').noteVersionPartial, 'list'),
            4: () => relPartial(require('./project').projectPartial, 'list'),
            5: () => relPartial(require('./projectVersion').projectVersionPartial, 'list'),
            6: () => relPartial(require('./routine').routinePartial, 'list'),
            7: () => relPartial(require('./routineVersion').routineVersionPartial, 'list'),
            8: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            9: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'list'),
            10: () => relPartial(require('./standard').standardPartial, 'list'),
            11: () => relPartial(require('./standardVersion').standardVersionPartial, 'list'),
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
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
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