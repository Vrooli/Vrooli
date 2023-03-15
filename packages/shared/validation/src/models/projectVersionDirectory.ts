import { description, id, name, req, opt, YupModel, blankToUndefined, maxStrErr, transRel, bool, yupObj } from '../utils';
import * as yup from 'yup';

export const childOrder = yup.string().transform(blankToUndefined).max(4096, maxStrErr)

export const projectVersionDirectoryTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
})

export const projectVersionDirectoryValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ['parentDirectory', ['Connect'], 'one', 'opt'],
        ['projectVersion', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', projectVersionDirectoryTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(bool),
    }, [
        ['parentDirectory', ['Connect', 'Disconnect'], 'one', 'opt'],
        ['projectVersion', ['Connect'], 'one', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionDirectoryTranslationValidation],
    ], [], o),
}