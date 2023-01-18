import { description, id, name, language, req, opt, YupModel, rel, blankToUndefined, maxStrErr } from '../utils';
import * as yup from 'yup';

export const childOrder = yup.string().transform(blankToUndefined).max(4096, maxStrErr)

export const projectVersionDirectoryTranslationValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        language: req(language),
        description: opt(description),
        name: req(name),
    }),
    update: () => yup.object().shape({
        id: req(id),
        language: opt(language),
        description: opt(description),
        name: opt(name),
    })
}

export const projectVersionDirectoryValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(yup.boolean()),
        ...rel('parentDirectory', ['Connect'], 'one', 'opt'),
        ...rel('projectVersion', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', projectVersionDirectoryTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        childOrder: opt(childOrder),
        isRoot: opt(yup.boolean()),
        ...rel('parentDirectory', ['Connect', 'Disconnect'], 'one', 'opt'),
        ...rel('projectVersion', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', projectVersionDirectoryTranslationValidation),
    }),
}