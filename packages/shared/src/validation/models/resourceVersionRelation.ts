/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in resourceVersionRelation.test.ts
import * as yup from "yup";
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { LABEL_MAX_LENGTH } from "../utils/validationConstants.js";

const label = yup.string().trim().removeEmptyString().max(LABEL_MAX_LENGTH, maxStrErr);
const labelsArray = yup.array().transform((value) => {
    // Pre-process the array to filter out empty strings before individual validation
    if (Array.isArray(value)) {
        const filtered = value.filter((item) => item !== "" && item !== null && item !== undefined);
        return filtered; // Return the filtered array, even if empty
    }
    return value;
}).of(label);

export const resourceVersionRelationValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        labels: labelsArray.notRequired().nullable(),
    }, [
        ["toVersion", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        labels: labelsArray.notRequired().nullable(),
    }, [], [], d),
};
/* c8 ignore stop */
