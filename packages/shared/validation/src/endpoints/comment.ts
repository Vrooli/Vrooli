import { blankToUndefined, id, language, maxStrErr, minStrErr, opt, rel, req } from '../base';
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

export const commentTranslationValidation = {
    create: yup.object().shape({
        id: req(id),
        language: req(language),
        text: req(text),
    }),
    update: yup.object().shape({
        id: req(id),
        language: opt(language),
        text: opt(text),
    })
}

export const commentValidation = {
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