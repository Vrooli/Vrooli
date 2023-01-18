// import { blankToUndefined, id, maxStrErr, req } from '../utils';
// import * as yup from 'yup';

// const data = yup.string().transform(blankToUndefined).max(8192, maxStrErr);

// export const runInputCreate = yup.object().shape({
//     id: req(id),
//     data: req(data),
//     inputId: req(id),
// })

// export const runInputUpdate = yup.object().shape({
//     id: req(id),
//     data: req(data),
// })

import * as yup from 'yup';
import { YupModel } from "../utils";

export const runRoutineInputValidation: YupModel = {
    create: () => yup.object().shape({
    }),
    update: () => yup.object().shape({
    }),
}