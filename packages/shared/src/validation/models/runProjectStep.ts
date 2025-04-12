import { RunStepStatus } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, intPositiveOrOne, intPositiveOrZero, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

const runStepStatus = enumToYup(RunStepStatus);

export const runProjectStepValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        complexity: req(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrOne),
        directoryInId: req(id),
        name: req(name),
        order: req(intPositiveOrZero),
        status: opt(runStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["directory", ["Connect"], "one", "opt"],
        ["runProject", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], d),
};
