import { NoteVersion, NoteVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
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
        pullRequest: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'full'),
        root: async () => relPartial((await import('./note')).notePartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        root: async () => relPartial((await import('./note')).notePartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(noteVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => relPartial((await import('./note')).notePartial, 'nav'),
        translations: () => relPartial(noteVersionTranslationPartial, 'list'),
    }
}