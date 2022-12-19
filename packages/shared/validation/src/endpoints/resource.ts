import { adaHandleRegex, blankToUndefined, description, id, index, maxStrErr, minNumErr, name, opt, rel, req, transRel, urlRegex, walletAddressRegex, YupModel } from '../utils';
import * as yup from 'yup';
import { ResourceUsedFor } from '@shared/consts';

// Link must match one of the regex above
const link = yup.string().transform(blankToUndefined).max(1024, maxStrErr).test(
    'link',
    'Must be a URL, Cardano payment address, or ADA Handle',
    (value: string | undefined) => {
        return value !== undefined ? (urlRegex.test(value) || walletAddressRegex.test(value) || adaHandleRegex.test(value)) : true
    }
)
const usedFor = yup.string().transform(blankToUndefined).oneOf(Object.values(ResourceUsedFor))

export const resourceTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: opt(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
})

export const resourceValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        index: opt(index),
        link: req(link),
        usedFor: opt(usedFor),
        ...rel('list', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'one', 'opt', resourceTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        index: opt(index),
        link: opt(link),
        usedFor: opt(usedFor),
        ...rel('list', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'one', 'opt', resourceTranslationValidation),
    }),
}