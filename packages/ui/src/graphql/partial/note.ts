import { Note, NoteYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { DeepPartialBooleanWithFragments, GqlPartial, NonMaybe } from "types";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

export const noteYouPartial: GqlPartial<NoteYou> = {
    __typename: 'NoteYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    }),
}

export const notePartial: GqlPartial<Note> = {
    __typename: 'Note',
    common: () => ({
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: relPartial(labelPartial, 'list'),
        owner: {
            __union: {
                Organization: relPartial(organizationPartial, 'nav'),
                User: relPartial(userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: relPartial(tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: relPartial(noteYouPartial, 'full'),
    }),
    full: () => ({
        versions: relPartial(noteVersionPartial, 'full', { omit: 'root' }),
    }),
    list: () => ({
        versions: relPartial(noteVersionPartial, 'list'),
    })
}