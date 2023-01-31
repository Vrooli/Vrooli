import { NoteVersion, NoteVersionTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";
import { versionYou } from "./root";

export const noteVersionTranslation: GqlPartial<NoteVersionTranslation> = {
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

export const noteVersion: GqlPartial<NoteVersion> = {
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
        you: () => rel(versionYou, 'full'),
    },
    full: {
        pullRequest: async () => rel((await import('./pullRequest')).pullRequest, 'full'),
        root: async () => rel((await import('./note')).note, 'full', { omit: 'versions' }),
        translations: () => rel(noteVersionTranslation, 'full'),
        versionNotes: true,
    },
    list: {
        root: async () => rel((await import('./note')).note, 'list', { omit: 'versions' }),
        translations: () => rel(noteVersionTranslation, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import('./note')).note, 'nav'),
        translations: () => rel(noteVersionTranslation, 'list'),
    }
}