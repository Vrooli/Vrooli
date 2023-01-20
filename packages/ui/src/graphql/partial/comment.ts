import { Comment, CommentTranslation, CommentYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiVersionPartial } from "./apiVersion";
import { issuePartial } from "./issue";
import { noteVersionPartial } from "./noteVersion";
import { organizationPartial } from "./organization";
import { postPartial } from "./post";
import { userPartial } from "./user";

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
                ApiVersion: () => relPartial(apiVersionPartial, 'nav'),
                Issue: () => relPartial(issuePartial, 'nav'),
                NoteVersion: () => relPartial(noteVersionPartial, 'nav'),
                Post: () => relPartial(postPartial, 'nav'),
                ProjectVersion: () => relPartial(projectVersionPartial, 'nav'),
                PullRequest: () => relPartial(pullRequestPartial, 'nav'),
                Question: () => relPartial(questionPartial, 'nav'),
                QuestionAnswer: () => relPartial(questionAnswerPartial, 'nav'),
                RoutineVersion: () => relPartial(routineVersionPartial, 'nav'),
                SmartContractVersion: () => relPartial(smartContractVersionPartial, 'nav'),
                StandardVersion: () => relPartial(standardVersionPartial, 'nav'),
            }
        },
        owner: {
            __union: {
                Organization: () => relPartial(organizationPartial, 'nav'),
                User: () => relPartial(userPartial, 'nav'),
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