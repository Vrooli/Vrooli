import { description, id, name, language, req, opt, YupModel, versionLabel, versionNotes, rel } from '../utils';
import * as yup from 'yup';
import { projectVersionDirectoryValidation } from './projectVersionDirectory';

export const projectVersionTranslationValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        language: req(language),
        description: opt(description),
        name: req(name),
    }),
    update: yup.object().shape({
        id: req(id),
        language: opt(language),
        description: opt(description),
        name: opt(name),
    })
}

export const projectVersionValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        isComplete: opt(yup.boolean()),
        versionIndex: req(yup.number()),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('translations', ['Create'], 'many', 'opt', projectVersionTranslationValidation),
        ...rel('directoryListings', ['Create'], 'many', 'opt', projectVersionDirectoryValidation),
        ...rel('suggestedNextByProject', ['Connect'], 'many', 'opt'),
    }),
    update: yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        isComplete: opt(yup.boolean()),
        versionIndex: opt(yup.number()),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionTranslationValidation),
        ...rel('directoryListings', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionDirectoryValidation),
        ...rel('suggestedNextByProject', ['Connect', 'Disconnect'], 'many', 'opt'),
    }),
}