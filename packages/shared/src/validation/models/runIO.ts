/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in runIO.test.ts
import * as yup from "yup";
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { NODE_INPUT_NAME_MAX_LENGTH, NODE_NAME_MAX_LENGTH, RUN_IO_DATA_MAX_LENGTH } from "../utils/validationConstants.js";

const data = yup.string().trim().removeEmptyString().max(RUN_IO_DATA_MAX_LENGTH, maxStrErr);
const nodeInputName = yup.string().trim().removeEmptyString().max(NODE_INPUT_NAME_MAX_LENGTH, maxStrErr);
const nodeName = yup.string().trim().removeEmptyString().max(NODE_NAME_MAX_LENGTH, maxStrErr);

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
/* c8 ignore stop */
