import { PullRequest, PullRequestYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const pullRequestYouPartial: GqlPartial<PullRequestYou> = {
    __typename: 'PullRequestYou',
    full: {
        canComment: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
    },
}

export const pullRequestPartial: GqlPartial<PullRequest> = {
    __typename: 'PullRequest',
    common: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./apiVersion').apiVersionPartial, 'list'],
            2: [require('./note').notePartial, 'list'],
            3: [require('./noteVersion').noteVersionPartial, 'list'],
            4: [require('./project').projectPartial, 'list'],
            5: [require('./projectVersion').projectVersionPartial, 'list'],
            6: [require('./routine').routinePartial, 'list'],
            7: [require('./routineVersion').routineVersionPartial, 'list'],
            8: [require('./smartContract').smartContractPartial, 'list'],
            9: [require('./smartContractVersion').smartContractVersionPartial, 'list'],
            10: [require('./standard').standardPartial, 'list'],
            11: [require('./standardVersion').standardVersionPartial, 'list'],
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
}