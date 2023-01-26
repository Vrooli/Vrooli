import { StandardVersion, StandardVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";
import { versionYouPartial } from "./root";

export const standardVersionTranslationPartial: GqlPartial<StandardVersionTranslation> = {
    __typename: 'StandardVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
    },
    full: {},
    list: {},
}

export const standardVersionPartial: GqlPartial<StandardVersion> = {
    __typename: 'StandardVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isComplete: true,
        isFile: true,
        isLatest: true,
        isPrivate: true,
        default: true,
        standardType: true,
        props: true,
        yup: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        versionNotes: true,
        pullRequest: () => relPartial(require('./pullRequest').pullRequestPartial, 'full'),
        resourceList: () => relPartial(require('./resourceList').resourceListPartial, 'full'),
        root: () => relPartial(require('./standard').standardPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(standardVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(standardVersionTranslationPartial, 'list'),
    }
}