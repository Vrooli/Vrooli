import * as yup from "yup";
import { bool, id, opt, req, YupModel, yupObj } from "../utils";

const creditsUsedBeforeLimit = yup.number().integer().min(0).max(1000000);
const absoluteMax = yup.number().integer().min(0).max(1000000);

export const apiKeyValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        creditsUsedBeforeLimit: req(creditsUsedBeforeLimit),
        stopAtLimit: req(bool),
        absoluteMax: req(absoluteMax),
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        creditsUsedBeforeLimit: opt(creditsUsedBeforeLimit),
        stopAtLimit: opt(bool),
        absoluteMax: opt(absoluteMax),
    }, [], [], o),
};
