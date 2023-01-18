import { blankToUndefined, description, id, name, opt, rel, req, transRel, YupModel } from '../utils';
import * as yup from 'yup';
import { labelValidation } from './label';

const issueFor = yup.string().transform(blankToUndefined).oneOf([
    'Api',
    'Organization',
    'Project',
    'Routine',
    'SmartContract',
    'Standard',
]);

export const issueTranslationValidation: YupModel = transRel({
    create: {
        name: req(name),
        description: opt(description),
    },
    update: {
        name: opt(name),
        description: opt(description),
    },
})

export const issueValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        issueFor: req(issueFor),
        ...rel('for', ['Connect'], 'one', 'req'),
        ...rel('labels', ['Connect', 'Create'], 'one', 'opt', labelValidation),
        ...rel('translations', ['Create'], 'many', 'opt', issueTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        ...rel('labels', ['Connect', 'Create', 'Disconnect'], 'one', 'opt', labelValidation),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', issueTranslationValidation),
    }),
}