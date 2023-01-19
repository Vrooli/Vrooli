import { PullRequestYou } from "@shared/consts";
import { GqlPartial } from "types";

export const pullRequestYouPartial: GqlPartial<PullRequestYou> = {
    __typename: 'PullRequestYou',
    full: () => ({
        canComment: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
    }),
}

export const listPullRequestFields = ['PullRequest', `{
    id
}`] as const;
export const pullRequestFields = ['PullRequest', `{
    id
}`] as const;