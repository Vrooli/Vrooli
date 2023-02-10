import { bool, details, id, opt, req, summary, transRel, url, versionLabel, versionNotes, YupModel, yupObj } from '../utils';
import { resourceListValidation } from './resourceList';
import { apiValidation } from './api';

export const apiVersionTranslationValidation: YupModel = transRel({
    create: {
        details: req(details),
        summary: opt(summary),
    },
    update: {
        details: opt(details),
        summary: opt(summary),
    },
})

export const apiVersionValidation: YupModel = {
    create: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['root', ['Connect', 'Create'], 'one', 'req', apiValidation, ['versions']],
        ['resourceList', ['Create'], 'one', 'opt', resourceListValidation],
        ['directoryListings', ['Connect'], 'many', 'opt'],
        ['translations', ['Create'], 'many', 'opt', apiVersionTranslationValidation],
    ], [['rootConnect', 'rootCreate']], o),
    update: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation],
        ['directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', apiVersionTranslationValidation],
    ], [], o),
}