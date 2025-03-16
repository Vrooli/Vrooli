import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bigIntString, bool, id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

const absoluteMax = yup.number().integer().min(0).max(1000000);

export const apiKeyValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        disabled: opt(bool),
        limitHard: req(bigIntString),
        limitSoft: opt(bigIntString),
        name: req(name),
        stopAtLimit: req(bool),
        absoluteMax: req(absoluteMax),
    }, [], [], d),
    update: (d) => yupObj({
        id: req(id),
        disabled: opt(bool),
        limitHard: opt(bigIntString),
        limitSoft: opt(bigIntString),
        name: opt(name),
        stopAtLimit: opt(bool),
        absoluteMax: opt(absoluteMax),
    }, [], [], d),
};
