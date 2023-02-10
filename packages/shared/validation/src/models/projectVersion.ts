import { description, id, name, req, opt, YupModel, versionLabel, versionNotes, rel, transRel, index, yupObj, bool } from '../utils';
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
    create: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        isComplete: opt(bool),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['root', ['Connect', 'Create'], 'one', 'req', projectValidation, ['versions']],
        ['translations', ['Create'], 'many', 'opt', projectVersionTranslationValidation],
        ['directoryListings', ['Create'], 'many', 'opt', projectVersionDirectoryValidation],
        ['suggestedNextByProject', ['Connect'], 'many', 'opt'],
    ], [['rootConnect', 'rootCreate']], o),
    update: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        isComplete: opt(bool),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionTranslationValidation],
        ['directoryListings', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionDirectoryValidation],
        ['suggestedNextByProject', ['Connect', 'Disconnect'], 'many', 'opt'],
    ], [], o),
}