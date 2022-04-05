import { id, name } from './base';
import * as yup from 'yup';

export const walletUpdate = yup.object().shape({
    id: id.required(),
    name: name.notRequired().default(undefined),
})