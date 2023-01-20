import { ApiVersion, ApiVersionTranslation } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { pullRequestPartial } from "./pullRequest";
import { versionYouPartial } from "./root";

export const apiVersionTranslationPartial: GqlPartial<ApiVersionTranslation> = {
    __typename: 'ApiVersionTranslation',
    full: {
        id: true,
        language: true,
        details: true,
        summary: true,
    },
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
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        root: () => relPartial(apiPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        root: () => relPartial(apiPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(apiVersionTranslationPartial, 'list'),
    }
}