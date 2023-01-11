import * as yup from 'yup';
import { description, jsonVariable, opt, transRel, YupModel } from "../utils";

export const standardVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
    },
    update: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
    }
})

export const standardVersionValidation: YupModel = {
    create: () => yup.object().shape({
    }),
    update: () => yup.object().shape({
    }),
}