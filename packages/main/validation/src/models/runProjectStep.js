import { RunProjectStepStatus } from "@local/consts";
import * as yup from "yup";
import { enumToYup, id, intPositiveOrOne, intPositiveOrZero, name, opt, req, yupObj } from "../utils";
const runProjectStepStatus = enumToYup(RunProjectStepStatus);
export const runProjectStepValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        name: req(name),
        order: req(intPositiveOrZero),
        step: req(yup.array().of(intPositiveOrZero)),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["directory", ["Connect"], "one", "opt"],
        ["node", ["Connect"], "one", "opt"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runProjectStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], o),
};
//# sourceMappingURL=runProjectStep.js.map