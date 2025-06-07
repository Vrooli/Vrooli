/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in apiKeyExternal.test.ts
import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { API_KEY_EXTERNAL_MAX_LENGTH, API_KEY_SERVICE_MAX_LENGTH } from "../utils/validationConstants.js";

const key = yup.string().trim().removeEmptyString().max(API_KEY_EXTERNAL_MAX_LENGTH, maxStrErr);
const service = yup.string().trim().removeEmptyString().max(API_KEY_SERVICE_MAX_LENGTH, maxStrErr);

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
