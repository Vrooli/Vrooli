import * as yup from "yup";
import { RunProjectStepStatus } from "../../api/types";
import { enumToYup, id, intPositiveOrOne, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";

const runProjectStepStatus = enumToYup(RunProjectStepStatus);

export const runProjectStepValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        name: req(name),
        order: req(intPositiveOrZero),
        step: req(yup.array().of(intPositiveOrZero)),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["directory", ["Connect"], "one", "opt"],
        ["runProject", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        contextSwitches: opt(intPositiveOrOne),
        status: opt(runProjectStepStatus),
        timeElapsed: opt(intPositiveOrZero),
    }, [], [], d),
};
