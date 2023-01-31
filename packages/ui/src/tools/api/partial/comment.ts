import { Comment, CommentThread, CommentTranslation, CommentYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const commentTranslation: GqlPartial<CommentTranslation> = {
    __typename: 'CommentTranslation',
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
}

export const commentYou: GqlPartial<CommentYou> = {
    __typename: 'CommentYou',
    common: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReply: true,
        canReport: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const comment: GqlPartial<Comment> = {
    __typename: 'Comment',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'nav'),
            1: async () => relPartial((await import('./issue')).issuePartial, 'nav'),
            2: async () => relPartial((await import('./noteVersion')).noteVersionPartial, 'nav'),
            3: async () => relPartial((await import('./post')).postPartial, 'nav'),
            4: async () => relPartial((await import('./projectVersion')).projectVersionPartial, 'nav'),
            5: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'nav'),
            6: async () => relPartial((await import('./question')).questionPartial, 'nav'),
            7: async () => relPartial((await import('./questionAnswer')).questionAnswerPartial, 'nav'),
            8: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav'),
            9: async () => relPartial((await import('./smartContractVersion')).smartContractVersionPartial, 'nav'),
            10: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'nav'),
            11: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
            12: async () => relPartial((await import('./user')).userPartial, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        commentedOn: {
            __union: {
                ApiVersion: 0,
                Issue: 1,
                NoteVersion: 2,
                Post: 3,
                ProjectVersion: 4,
                PullRequest: 5,
                Question: 6,
                QuestionAnswer: 7,
                RoutineVersion: 8,
                SmartContractVersion: 9,
                StandardVersion: 10,
            }
        },
        owner: {
            __union: {
                Organization: 11,
                User: 12,
            }
        },
        score: true,
        stars: true,
        reportsCount: true,
        you: () => relPartial(commentYou, 'full'),
    },
    full: {
        translations: () => relPartial(commentTranslation, 'full'),
    },
    list: {
        translations: () => relPartial(commentTranslation, 'list'),
    }
}

export const commentThreadPartial: GqlPartial<CommentThread> = {
    __typename: 'CommentThread',
    common: {
        childThreads: {
            childThreads: {
                comment: () => relPartial(comment, 'list'),
                endCursor: true,
                totalInThread: true,
            },
            comment: () => relPartial(comment, 'list'),
            endCursor: true,
            totalInThread: true,
        },
        comment: () => relPartial(comment, 'list'),
        endCursor: true,
        totalInThread: true,
    },
}