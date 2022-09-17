import { id, name, requiredErrorMessage } from './base';
import * as yup from 'yup';

export const walletUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    name: name.notRequired().default(undefined),
})

export const walletsUpdate = yup.array().of(walletUpdate.required(requiredErrorMessage))