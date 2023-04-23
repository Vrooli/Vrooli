import { RunRoutineStepStatus } from "@local/consts";
import * as yup from "yup";
import { enumToYup, id, intPositiveOrOne, intPositiveOrZero, name, opt, req, yupObj } from "../utils";
const runRoutineStepStatus = enumToYup(RunRoutineStepStatus);
export const runRoutineStepValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        name: req(name),
        order: req(intPositiveOrZero),
        step: req(yup.array().of(intPositiveOrZero)),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["node", ["Connect"], "one", "opt"],
        ["subroutineVersion", ["Connect"], "one", "opt"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runRoutineStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], o),
};
//# sourceMappingURL=runRoutineStep.js.map