import { details, id, index, opt, rel, req, summary, transRel, versionLabel, versionNotes, YupModel } from '../utils';
import * as yup from 'yup';
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
    create: () => yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: req(index),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('root', ['Connect', 'Create'], 'one', 'req', noteValidation),
        ...rel('directoryListings', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', noteVersionTranslationValidation),
    }, [['rootConnect', 'rootCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: opt(index),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', noteVersionTranslationValidation),
    }),
}