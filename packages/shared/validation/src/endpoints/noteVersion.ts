import { details, id, opt, rel, req, summary, transRel, versionLabel, versionNotes, YupModel } from '../utils';
import * as yup from 'yup';

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
    create: yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: req(yup.number()),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('directoryListings', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', noteVersionTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: opt(yup.number()),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', noteVersionTranslationValidation),
    }),
}