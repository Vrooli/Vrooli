import { bool, details, id, opt, req, summary, transRel, versionLabel, versionNotes, YupModel, yupObj } from '../utils';
import { noteValidation } from './note';

export const noteVersionTranslationValidation: YupModel = transRel({
    create: {
        details: req(details),
        summary: opt(summary),
    },
    update: {
        details: opt(details),
        summary: opt(summary),
    },
})

export const noteVersionValidation: YupModel = {
    create: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['root', ['Connect', 'Create'], 'one', 'req', noteValidation, ['versions']],
        ['directoryListings', ['Connect'], 'many', 'opt'],
        ['translations', ['Create'], 'many', 'opt', noteVersionTranslationValidation],
    ], [['rootConnect', 'rootCreate']], o),
    update: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', noteVersionTranslationValidation],
    ], [], o),
}