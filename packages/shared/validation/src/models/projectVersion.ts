import { description, id, name, req, opt, YupModel, versionLabel, versionNotes, rel, transRel, index } from '../utils';
import * as yup from 'yup';
import { projectVersionDirectoryValidation } from './projectVersionDirectory';
import { projectValidation } from './project';

export const projectVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const projectVersionValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        isComplete: opt(yup.boolean()),
        versionIndex: req(index),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('root', ['Connect', 'Create'], 'one', 'req', projectValidation),
        ...rel('translations', ['Create'], 'many', 'opt', projectVersionTranslationValidation),
        ...rel('directoryListings', ['Create'], 'many', 'opt', projectVersionDirectoryValidation),
        ...rel('suggestedNextByProject', ['Connect'], 'many', 'opt'),
    }, [['rootConnect', 'rootCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        isComplete: opt(yup.boolean()),
        versionIndex: opt(index),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionTranslationValidation),
        ...rel('directoryListings', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionDirectoryValidation),
        ...rel('suggestedNextByProject', ['Connect', 'Disconnect'], 'many', 'opt'),
    }),
}