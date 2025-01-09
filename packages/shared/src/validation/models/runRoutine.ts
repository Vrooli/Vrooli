import { RunStatus } from "../../api/types";
import { bool, enumToYup, id, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";
import { runRoutineInputValidation } from "./runRoutineInput";
import { runRoutineOutputValidation } from "./runRoutineOutput";
import { runRoutineStepValidation } from "./runRoutineStep";
import { scheduleValidation } from "./schedule";

const runStatus = enumToYup(RunStatus);

export const runRoutineValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        status: req(runStatus),
        name: req(name),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["inputs", ["Create"], "many", "opt", runRoutineInputValidation],
        ["outputs", ["Create"], "many", "opt", runRoutineOutputValidation],
        ["routineVersion", ["Connect"], "one", "req"],
        ["runProject", ["Connect"], "one", "opt"],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["steps", ["Create"], "many", "opt", runRoutineStepValidation],
        ["team", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        isStarted: opt(bool),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["inputs", ["Create", "Update", "Delete"], "many", "opt", runRoutineInputValidation],
        ["outputs", ["Create", "Update", "Delete"], "many", "opt", runRoutineOutputValidation],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["steps", ["Create", "Update", "Delete"], "many", "opt", runRoutineStepValidation],
    ], [], d),
};
