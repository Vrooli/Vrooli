import { Note, NoteYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const noteYouPartial: GqlPartial<NoteYou> = {
    __typename: 'NoteYou',
    full: {
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
}

export const notePartial: GqlPartial<Note> = {
    __typename: 'Note',
    common: {
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: () => relPartial(require('./label').labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
                User: () => relPartial(require('./user').userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(require('./tag').tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: () => relPartial(noteYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./noteVersion').noteVersionPartial, 'full', { omit: 'root' }),
    },
    list: {
        versions: () => relPartial(require('./noteVersion').noteVersionPartial, 'list'),
    }
}