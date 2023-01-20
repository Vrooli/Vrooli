import { NoteVersion, NoteVersionTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { notePartial } from "./note";
import { pullRequestPartial } from "./pullRequest";
import { versionYouPartial } from "./root";

export const noteVersionTranslationPartial: GqlPartial<NoteVersionTranslation> = {
    __typename: 'NoteVersionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        text: true,
    },
}

export const noteVersionPartial: GqlPartial<NoteVersion> = {
    __typename: 'NoteVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        versionIndex: true,
        versionLabel: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        root: () => relPartial(notePartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        root: () => relPartial(notePartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'list'),
    }
}