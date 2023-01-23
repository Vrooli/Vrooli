import { Comment, CommentTranslation, CommentYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const commentTranslationPartial: GqlPartial<CommentTranslation> = {
    __typename: 'CommentTranslation',
    full: {
        id: true,
        language: true,
        text: true,
    },
}

export const commentYouPartial: GqlPartial<CommentYou> = {
    __typename: 'CommentYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReply: true,
        canReport: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    },
}

export const commentPartial: GqlPartial<Comment> = {
    __typename: 'Comment',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        commentedOn: {
            __union: {
                ApiVersion: () => relPartial(require('./apiVersion').apiVersionPartial, 'nav'),
                Issue: () => relPartial(require('./issue').issuePartial, 'nav'),
                NoteVersion: () => relPartial(require('./noteVersion').noteVersionPartial, 'nav'),
                Post: () => relPartial(require('./post').postPartial, 'nav'),
                ProjectVersion: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav'),
                PullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'nav'),
                Question: () => relPartial(require('./question').questionPartial, 'nav'),
                QuestionAnswer: () => relPartial(require('./questionAnswer').questionAnswerPartial, 'nav'),
                RoutineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
                SmartContractVersion: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'nav'),
                StandardVersion: () => relPartial(require('./standardVersion').standardVersionPartial, 'nav'),
            }
        },
        owner: {
            __union: {
                Organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
                User: () => relPartial(require('./user').userPartial, 'nav'),
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