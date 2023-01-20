import { PullRequest, PullRequestYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { apiVersionPartial } from "./apiVersion";
import { notePartial } from "./note";
import { noteVersionPartial } from "./noteVersion";
import { projectPartial } from "./project";
import { projectVersionPartial } from "./projectVersion";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";
import { userPartial } from "./user";

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
            0: [apiPartial, 'list'],
            1: [apiVersionPartial, 'list'],
            2: [notePartial, 'list'],
            3: [noteVersionPartial, 'list'],
            4: [projectPartial, 'list'],
            5: [projectVersionPartial, 'list'],
            6: [routinePartial, 'list'],
            7: [routineVersionPartial, 'list'],
            8: [smartContractPartial, 'list'],
            9: [smartContractVersionPartial, 'list'],
            10: [standardPartial, 'list'],
            11: [standardVersionPartial, 'list'],
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
        createdBy: () => relPartial(userPartial, 'nav'),
        you: () => relPartial(pullRequestYouPartial, 'full'),
    },
}