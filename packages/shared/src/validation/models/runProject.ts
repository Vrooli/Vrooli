import { RunStatus } from "@local/shared";
import { bool, enumToYup, id, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";
import { runProjectStepValidation } from "./runProjectStep";
import { scheduleValidation } from "./schedule";

const runStatus = enumToYup(RunStatus);

export const runProjectValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        status: req(runStatus),
        name: req(name),
    }, [
        ["organization", ["Connect"], "one", "opt"],
        ["projectVersion", ["Connect"], "one", "req"],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["steps", ["Create"], "many", "opt", runProjectStepValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        isStarted: opt(bool),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["steps", ["Create", "Update", "Delete"], "many", "opt", runProjectStepValidation],
    ], [], d),
};
