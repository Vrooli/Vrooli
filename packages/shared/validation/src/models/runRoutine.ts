import { bool, enumToYup, id, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";
import { RunStatus } from "@shared/consts";
import { runRoutineStepValidation } from "./runRoutineStep";
import { runRoutineScheduleValidation } from "./runRoutineSchedule";
import { runRoutineInputValidation } from "./runRoutineInput";

const runStatus = enumToYup(RunStatus);

export const runRoutineValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        status: req(runStatus),
        name: req(name),
    }, [
        ['inputs', ['Create'], 'many', 'opt', runRoutineInputValidation],
        ['organization', ['Connect'], 'one', 'opt'],
        ['routineVersion', ['Connect'], 'one', 'req'],
        ['runProject', ['Connect'], 'one', 'opt'],
        ['runRoutineSchedule', ['Connect', 'Create'], 'many', 'opt', runRoutineScheduleValidation],
        ['steps', ['Create'], 'many', 'opt', runRoutineStepValidation],
    ], [['runRoutineScheduleConnect', 'runRoutineScheduleCreate']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        isStarted: opt(bool),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ['inputs', ['Create', 'Update', 'Delete'], 'many', 'opt', runRoutineInputValidation],
        ['runRoutineSchedule', ['Connect', 'Create'], 'many', 'opt', runRoutineScheduleValidation],
        ['steps', ['Create', 'Update', 'Delete'], 'many', 'opt', runRoutineStepValidation],
    ], [['runRoutineScheduleConnect', 'runRoutineScheduleCreate']], o),
}