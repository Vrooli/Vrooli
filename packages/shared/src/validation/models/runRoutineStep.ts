import { RunRoutineStepStatus } from "@local/shared";
import * as yup from "yup";
import { enumToYup, id, intPositiveOrOne, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";

const runRoutineStepStatus = enumToYup(RunRoutineStepStatus);

export const runRoutineStepValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        name: req(name),
        order: req(intPositiveOrZero),
        step: req(yup.array().of(intPositiveOrZero)),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["node", ["Connect"], "one", "opt"],
        ["subroutineVersion", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runRoutineStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], d),
};
