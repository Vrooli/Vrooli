/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in runStep.test.ts
import * as yup from "yup";
import { RunStepStatus } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, intPositiveOrOne, intPositiveOrZero, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

const nodeId = yup.string().trim().removeEmptyString().max(128, maxStrErr);
const runStepStatus = enumToYup(RunStepStatus);

export const runRoutineStepValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        complexity: req(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrOne),
        name: req(name),
        nodeId: req(nodeId),
        order: req(intPositiveOrZero),
        status: opt(runStepStatus),
        resourceInId: req(id),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["run", ["Connect"], "one", "req"],
        ["resourceVersion", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], d),
};
/* c8 ignore stop */
