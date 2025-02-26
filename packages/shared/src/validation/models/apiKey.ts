import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

const creditsUsedBeforeLimit = yup.number().integer().min(0).max(1000000);
const absoluteMax = yup.number().integer().min(0).max(1000000);

export const apiKeyValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        creditsUsedBeforeLimit: req(creditsUsedBeforeLimit),
        stopAtLimit: req(bool),
        absoluteMax: req(absoluteMax),
    }, [], [], d),
    update: (d) => yupObj({
        id: req(id),
        creditsUsedBeforeLimit: opt(creditsUsedBeforeLimit),
        stopAtLimit: opt(bool),
        absoluteMax: opt(absoluteMax),
    }, [], [], d),
};
