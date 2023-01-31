import { Note, NoteYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const noteYouPartial: GqlPartial<NoteYou> = {
    __typename: 'NoteYou',
    common: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
    full: {},
    list: {},
}

export const notePartial: GqlPartial<Note> = {
    __typename: 'Note',
    common: {
        __define: {
            0: () => relPartial(require('./organization').organizationPartial, 'nav'),
            1: () => relPartial(require('./user').userPartial, 'nav'),
            2: () => relPartial(require('./tag').tagPartial, 'list'),
            3: () => relPartial(require('./label').labelPartial, 'list'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: { __use: 3 },
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: { __use: 2 },
        transfersCount: true,
        views: true,
        you: () => relPartial(noteYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./noteVersion').noteVersionPartial, 'full', { omit: 'root' }),
    },
    list: {
        versions: () => relPartial(require('./noteVersion').noteVersionPartial, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isPrivate: true,
    }
}