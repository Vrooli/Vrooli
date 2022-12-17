import * as yup from 'yup';
import { id, opt, req, YupModel } from "utils";

const creditsUsedBeforeLimit = yup.number().integer().min(0).max(1000000);
const absoluteMax = yup.number().integer().min(0).max(1000000);

export const apiKeyValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        creditsUsedBeforeLimit: req(creditsUsedBeforeLimit),
        stopAtLimit: req(yup.boolean()),
        absoluteMax: req(absoluteMax),
    }),
    update: yup.object().shape({
        id: req(id),
        creditsUsedBeforeLimit: opt(creditsUsedBeforeLimit),
        stopAtLimit: opt(yup.boolean()),
        absoluteMax: opt(absoluteMax),
    }),
}