import { adaHandleRegex, blankToUndefined, description, enumToYup, id, index, maxStrErr, name, opt, req, transRel, urlRegex, walletAddressRegex, YupModel, yupObj } from '../utils';
import * as yup from 'yup';
import { ResourceUsedFor } from '@shared/consts';

// Link must match one of the regex above
const link = yup.string().transform(blankToUndefined).max(1024, maxStrErr).test(
    'link',
    'Must be a URL, Cardano payment address, or ADA Handle',
    (value: string | undefined) => {
        console.log('in link test!', )
        return value !== undefined ? (urlRegex.test(value) || walletAddressRegex.test(value) || adaHandleRegex.test(value)) : true
    }
)
const usedFor = enumToYup(ResourceUsedFor);

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
    create: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        link: req(link),
        usedFor: opt(usedFor),
    }, [
        ['list', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'one', 'opt', resourceTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        link: opt(link),
        usedFor: opt(usedFor),
    }, [
        ['list', ['Connect'], 'one', 'req'],
        ['translations', ['Create', 'Update', 'Delete'], 'one', 'opt', resourceTranslationValidation],
    ], [], o),
}