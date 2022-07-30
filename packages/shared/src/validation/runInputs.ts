import { id } from './base';
import * as yup from 'yup';

const data = yup.string().max(8192);

export const runInputCreate = yup.object().shape({
    id: id.required(),
    data: data.required(),
})

export const runInputUpdate = yup.object().shape({
    id: id.required(),
    data: data.required(),
})

export const runInputsCreate = yup.array().of(runInputCreate.required())
export const runInputsUpdate = yup.array().of(runInputUpdate.required())