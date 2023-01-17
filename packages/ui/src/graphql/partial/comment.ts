import { organizationNameFields } from "./organization";
import { projectNameFields } from "./project";
import { routineNameFields } from "./routine";
import { standardNameFields } from "./standard";
import { userNameFields } from "./user";

export const commentFields = ['Comment', `{
    id
    created_at
    updated_at
    score
    isUpvoted
    isStarred
    commentedOn {
        ... on Project ${projectNameFields}
        ... on Routine ${routineNameFields}
        ... on Standard ${standardNameFields}
    }
    creator {
        ... on Organization ${organizationNameFields}
        ... on User ${userNameFields}
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