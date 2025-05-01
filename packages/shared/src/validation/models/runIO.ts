import * as yup from "yup";
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const data = yup.string().trim().removeEmptyString().max(8192, maxStrErr);
const nodeInputName = yup.string().trim().removeEmptyString().max(128, maxStrErr);
const nodeName = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const runIOValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        data: req(data),
        nodeInputName: req(nodeInputName),
        nodeName: req(nodeName),
    }, [
        ["run", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        data: req(data),
    }, [], [], d),
};
