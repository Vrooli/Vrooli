import { CommentYou } from "@shared/consts";
import { GqlPartial } from "types";
import { organizationPartial } from "./organization";
import { projectNameFields } from "./project";
import { routineNameFields } from "./routine";
import { standardNameFields } from "./standard";
import { userNameFields } from "./user";

export const commentYouPartial: GqlPartial<CommentYou> = {
    __typename: 'CommentYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReply: true,
        canReport: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    }),
}

export const commentFields = ['Comment', `{
    id
    created_at
    updated_at
    score
    isUpvoted
    isStarred
    commentedOn {
        ... on Project ${projectNameFields[1]}
        ... on Routine ${routineNameFields[1]}
        ... on Standard ${standardNameFields[1]}
    }
    creator {
        ... on Organization ${organizationPartial.nav}
        ... on User ${userPartial.nav}
    }
    permissionsComment {
        canDelete
        canEdit
        canStar
        canReply
        canReport
        canVote
    }
    translations {
        id
        language
        text
    }
}`] as const;
export const commentThreadFields = ['CommentThread', `{
    childThreads {
        childThreads {
            comment ${commentFields[1]}
            endCursor
            totalInThread
        }
        comment ${commentFields[1]}
        endCursor
        totalInThread
    }
    comment ${commentFields[1]}
    endCursor
    totalInThread
}`] as const;