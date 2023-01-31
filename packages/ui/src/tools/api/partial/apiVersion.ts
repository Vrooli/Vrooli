import { ApiVersion, ApiVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
import { versionYouPartial } from "./root";

export const apiVersionTranslationPartial: GqlPartial<ApiVersionTranslation> = {
    __typename: 'ApiVersionTranslation',
    common: {
        id: true,
        language: true,
        details: true,
        summary: true,
    },
    full: {},
    list: {},
}

export const apiVersionPartial: GqlPartial<ApiVersion> = {
    __typename: 'ApiVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        callLink: true,
        commentsCount: true,
        documentationLink: true,
        forksCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        versionIndex: true,
        versionLabel: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        pullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'full'),
        root: () => relPartial(require('./api').apiPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        root: () => relPartial(require('./api').apiPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: () => relPartial(require('./api').apiPartial, 'nav'),
        translations: () => relPartial(apiVersionTranslationPartial, 'list'),
    }
}