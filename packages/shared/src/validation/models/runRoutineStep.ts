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
        subroutineInId: req(id),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["runRoutine", ["Connect"], "one", "req"],
        ["subroutine", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], d),
};
