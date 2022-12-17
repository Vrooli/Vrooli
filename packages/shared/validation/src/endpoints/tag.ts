import { description, id, language, opt, rel, req, tag, YupModel } from '../utils';
import * as yup from 'yup';

const anonymous = yup.boolean() // Determines if the user will be credited for the tag

export const tagTranslationValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        language: req(language),
        description: req(description),
    }),
    update: yup.object().shape({
        id: req(id),
        language: req(language),
        description: req(description),
    })
}

export const tagValidation: YupModel = {
    create: yup.object().shape({
        anonymous: opt(anonymous),
        tag: req(tag),
        ...rel('translations', ['Create'], 'many', 'opt', tagTranslationValidation),
    }),
    update: yup.object().shape({
        anonymous: opt(anonymous),
        tag: req(tag),
        ...rel('translations', ['Delete', 'Create', 'Update'], 'many', 'opt', tagTranslationValidation),
    }),
}