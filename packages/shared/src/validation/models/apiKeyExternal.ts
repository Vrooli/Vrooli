import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const key = yup.string().trim().removeEmptyString().max(255, maxStrErr);
const service = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const apiKeyExternalValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        disabled: opt(bool),
        key: req(key),
        name: req(name),
        service: req(service),
    }, [], [], d),
    update: (d) => yupObj({
        id: req(id),
        disabled: opt(bool),
        key: opt(key),
        name: opt(name),
        service: opt(service),
    }, [], [], d),
};
