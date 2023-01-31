import { SmartContractVersion, SmartContractVersionTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
import { versionYouPartial } from "./root";

export const smartContractVersionTranslationPartial: GqlPartial<SmartContractVersionTranslation> = {
    __typename: 'SmartContractVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
    },
    full: {},
    list: {},
}

export const smartContractVersionPartial: GqlPartial<SmartContractVersion> = {
    __typename: 'SmartContractVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        default: true,
        contractType: true,
        content: true,
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
        root: () => relPartial(require('./smartContract').smartContractPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(smartContractVersionTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(smartContractVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: () => relPartial(require('./smartContract').smartContractPartial, 'nav'),
        translations: () => relPartial(smartContractVersionTranslationPartial, 'list'),
    }
}