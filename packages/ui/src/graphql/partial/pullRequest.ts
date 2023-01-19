import { PullRequest, PullRequestYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { userPartial } from "./user";

export const pullRequestYouPartial: GqlPartial<PullRequestYou> = {
    __typename: 'PullRequestYou',
    full: () => ({
        canComment: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
    }),
}

export const pullRequestPartial: GqlPartial<PullRequest> = {
    __typename: 'PullRequest',
    common: () => ({
        __define: {
            0: [apiPartial, 'list'],
            1: [notePartial, 'list'],
            2: [projectPartial, 'list'],
            3: [routinePartial, 'list'],
            4: [smartContractPartial, 'list'],
            5: [standardPartial, 'list'],
        },
        id: true,
        created_at: true,
        updated_at: true,
        mergedOrRejectedAt: true,
        commentsCount: true,
        status: true,
        from: {
            __union: {
                Api: 0,
                Note:1,
                Project: 2,
                Routine: 3,
                SmartContract: 4,
                Standard: 5,
            }
        },
        to: {
            __union: {
                Api: 0,
                Note:1,
                Project: 2,
                Routine: 3,
                SmartContract: 4,
                Standard: 5,
            }
        },
        createdBy: relPartial(userPartial, 'nav'),
        you: relPartial(pullRequestYouPartial, 'full'),
    }),
}