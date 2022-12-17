import { details, id, opt, rel, req, summary, transRel, url, versionLabel, versionNotes, YupModel } from '../utils';
import * as yup from 'yup';
import { resourceListValidation } from './resourceList';

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
    create: yup.object().shape({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(yup.boolean()),
        versionIndex: req(yup.number()),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('resourceList', ['Create'], 'one', 'opt', resourceListValidation),
        ...rel('directoryListings', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', apiVersionTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(yup.boolean()),
        versionIndex: opt(yup.number()),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation),
        ...rel('directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', apiVersionTranslationValidation),
    }),
}