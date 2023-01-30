import { Comment, CommentThread, CommentTranslation, CommentYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const commentTranslationPartial: GqlPartial<CommentTranslation> = {
    __typename: 'CommentTranslation',
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
}

export const commentYouPartial: GqlPartial<CommentYou> = {
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

export const commentPartial: GqlPartial<Comment> = {
    __typename: 'Comment',
    common: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'nav'),
            1: () => relPartial(require('./issue').issuePartial, 'nav'),
            2: () => relPartial(require('./noteVersion').noteVersionPartial, 'nav'),
            3: () => relPartial(require('./post').postPartial, 'nav'),
            4: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav'),
            5: () => relPartial(require('./pullRequest').pullRequestPartial, 'nav'),
            6: () => relPartial(require('./question').questionPartial, 'nav'),
            7: () => relPartial(require('./questionAnswer').questionAnswerPartial, 'nav'),
            8: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
            9: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'nav'),
            10: () => relPartial(require('./standardVersion').standardVersionPartial, 'nav'),
            11: () => relPartial(require('./organization').organizationPartial, 'nav'),
            12: () => relPartial(require('./user').userPartial, 'nav'),
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
        you: () => relPartial(commentYouPartial, 'full'),
    },
    full: {
        translations: () => relPartial(commentTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(commentTranslationPartial, 'list'),
    }
}

export const commentThreadPartial: GqlPartial<CommentThread> = {
    __typename: 'CommentThread',
    common: {
        childThreads: {
            childThreads: {
                comment: () => relPartial(commentPartial, 'list'),
                endCursor: true,
                totalInThread: true,
            },
            comment: () => relPartial(commentPartial, 'list'),
            endCursor: true,
            totalInThread: true,
        },
        comment: () => relPartial(commentPartial, 'list'),
        endCursor: true,
        totalInThread: true,
    },
}