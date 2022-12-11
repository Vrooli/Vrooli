import { blankToUndefined, id, maxStrErr, req, reqArr } from './base';
import * as yup from 'yup';

const data = yup.string().transform(blankToUndefined).max(8192, maxStrErr);

export const runInputCreate = yup.object().shape({
    id: req(id),
    data: req(data),
    inputId: req(id),
})

export const runInputUpdate = yup.object().shape({
    id: req(id),
    data: req(data),
})

export const runInputsCreate = reqArr(runInputCreate)
export const runInputsUpdate = reqArr(runInputUpdate)