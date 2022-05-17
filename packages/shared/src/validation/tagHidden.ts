import { id } from './base';
import * as yup from 'yup';
import { tagCreate } from './tag';

const isBlur = yup.boolean();

export const tagHiddenCreate = yup.object().shape({
    isBlur: isBlur.notRequired().default(undefined),
    tagConnect: id.notRequired().default(undefined),
    tagCreate: tagCreate.notRequired().default(undefined),
}, [['tagConnect', 'tagCreate']]) // Cannot create and connect at the same time

export const tagHiddenUpdate = yup.object().shape({
    id: id.required(),
    isBlur: isBlur.notRequired().default(undefined),
})

export const tagHiddensCreate = yup.array().of(tagHiddenCreate.required())
export const tagHiddensUpdate = yup.array().of(tagHiddenUpdate.required())
