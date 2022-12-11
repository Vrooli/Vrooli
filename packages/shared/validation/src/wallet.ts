import { id, name, opt, req, reqArr } from './base';
import * as yup from 'yup';

export const walletUpdate = yup.object().shape({
    id: req(id),
    name: opt(name),
})

export const walletsUpdate = reqArr(walletUpdate)