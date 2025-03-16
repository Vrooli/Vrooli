import * as yup from "yup";
import { RunStatus } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, intPositiveOrZero, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";
import { runRoutineIOValidation } from "./runRoutineIO.js";
import { runRoutineStepValidation } from "./runRoutineStep.js";
import { scheduleValidation } from "./schedule.js";

const data = yup.string().trim().removeEmptyString().max(16384, maxStrErr);
const runStatus = enumToYup(RunStatus);

export const runRoutineValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        data: opt(data),
        isPrivate: opt(bool),
        status: req(runStatus),
        name: req(name),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["io", ["Create"], "many", "opt", runRoutineIOValidation],
        ["routineVersion", ["Connect"], "one", "req"],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["steps", ["Create"], "many", "opt", runRoutineStepValidation],
        ["team", ["Connect"], "one", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        data: opt(data),
        isPrivate: opt(bool),
        isStarted: opt(bool),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ["io", ["Create", "Update", "Delete"], "many", "opt", runRoutineIOValidation],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["steps", ["Create", "Update", "Delete"], "many", "opt", runRoutineStepValidation],
    ], [], d),
};
