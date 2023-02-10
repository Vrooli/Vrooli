import { bool, enumToYup, id, intPositiveOrZero, name, opt, req, YupModel, yupObj } from "../utils";
import { RunStatus } from "@shared/consts";
import { runProjectStepValidation } from "./runProjectStep";
import { runProjectScheduleValidation } from "./runProjectSchedule";

const runStatus = enumToYup(RunStatus);

export const runProjectValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        status: req(runStatus),
        name: req(name),
    }, [
        ['organization', ['Connect'], 'one', 'opt'],
        ['projectVersion', ['Connect'], 'one', 'req'],
        ['runProjectSchedule', ['Connect', 'Create'], 'many', 'opt', runProjectScheduleValidation],
        ['steps', ['Create'], 'many', 'opt', runProjectStepValidation],
    ], [['runProjectScheduleConnect', 'runProjectScheduleCreate']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        completedComplexity: opt(intPositiveOrZero),
        contextSwitches: opt(intPositiveOrZero),
        isPrivate: opt(bool),
        isStarted: opt(bool),
        timeElapsed: opt(intPositiveOrZero),
    }, [
        ['runProjectSchedule', ['Connect', 'Create'], 'many', 'opt', runProjectScheduleValidation],
        ['steps', ['Create', 'Update', 'Delete'], 'many', 'opt', runProjectStepValidation],
    ], [['runProjectScheduleConnect', 'runProjectScheduleCreate']], o),
}