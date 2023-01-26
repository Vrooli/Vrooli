import { NoteVersion, NoteVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";
import { versionYouPartial } from "./root";

export const noteVersionTranslationPartial: GqlPartial<NoteVersionTranslation> = {
    __typename: 'NoteVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        text: true,
    },
    full: {},
    list: {},
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
        pullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'full'),
        root: () => relPartial(require('./note').notePartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        root: () => relPartial(require('./note').notePartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'list'),
    }
}