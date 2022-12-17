import { blankToUndefined, id, maxStrErr, minStrErr, opt, rel, req, transRel, YupModel } from '../utils';
import * as yup from 'yup';

const createdFor = yup.string().transform(blankToUndefined).oneOf([
    'ApiVersion',
    'Issue',
    'NoteVersion',
    'Post',
    'ProjectVersion',
    'PullRequest',
    'Question',
    'QuestionAnswer',
    'RoutineVersion',
    'SmartContractVersion',
    'StandardVersion'
]);
const text = yup.string().transform(blankToUndefined).min(1, minStrErr).max(8192, maxStrErr)

export const commentTranslationValidation: YupModel = transRel({
    create: {
        text: req(text),
    },
    update: {
        text: opt(text),
    }
})

export const commentValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        createdFor: req(createdFor),
        forConnect: req(id),
        parentConnect: opt(id),
        ...rel('translations', ['Create'], 'many', 'opt', commentTranslationValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        ...rel('translations', ['Delete', 'Create', 'Update'], 'many', 'opt', commentTranslationValidation),
    }),
}