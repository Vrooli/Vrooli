import { details, id, index, opt, rel, req, summary, transRel, url, versionLabel, versionNotes, YupModel } from '../utils';
import * as yup from 'yup';
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
    create: () => yup.object().shape({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(yup.boolean()),
        versionIndex: req(index),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('root', ['Connect', 'Create'], 'one', 'req', apiValidation),
        ...rel('resourceList', ['Create'], 'one', 'opt', resourceListValidation),
        ...rel('directoryListings', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', apiVersionTranslationValidation),
    }, [['rootConnect', 'rootCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        callLink: opt(url),
        documentationLink: opt(url),
        isLatest: opt(yup.boolean()),
        versionIndex: opt(index),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation),
        ...rel('directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', apiVersionTranslationValidation),
    }),
}