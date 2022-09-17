import { id, maxStringErrorMessage, requiredErrorMessage } from './base';
import * as yup from 'yup';

const data = yup.string().max(8192, maxStringErrorMessage);

export const runInputCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    data: data.required(requiredErrorMessage),
    inputId: id.required(requiredErrorMessage),
})

export const runInputUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    data: data.required(requiredErrorMessage),
})

export const runInputsCreate = yup.array().of(runInputCreate.required(requiredErrorMessage))
export const runInputsUpdate = yup.array().of(runInputUpdate.required(requiredErrorMessage))