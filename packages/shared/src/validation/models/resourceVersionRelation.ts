/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in resourceVersionRelation.test.ts
import * as yup from "yup";
import { optArr, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const label = yup.string().trim().removeEmptyString().max(128, maxStrErr);

export const resourceVersionRelationValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        labels: optArr(label),
    }, [
        ["fromVersion", ["Connect"], "one", "req"],
        ["toVersion", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        labels: optArr(label),
    }, [], [], d),
};
/* c8 ignore stop */
