import { description, endTime, id, name, opt, req, startTime, timezone, transRel, YupModel, yupObj } from "../utils";
import { focusModeValidation } from "./focusMode";
import { labelValidation } from './label';
import { meetingValidation } from "./meeting";
import { runProjectValidation } from "./runProject";
import { runRoutineValidation } from "./runRoutine";
import { scheduleExceptionValidation } from "./scheduleException";
import { scheduleRecurrenceValidation } from "./scheduleRecurrence";

export const scheduleTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const scheduleValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: req(timezone),
    }, [
        ['exceptions', ['Create'], 'many', 'opt', scheduleExceptionValidation],
        ['focusModes', ['Connect'], 'many', 'opt', focusModeValidation],
        ['labels', ['Create', 'Connect'], 'many', 'opt', labelValidation],
        ['meetings', ['Connect'], 'many', 'opt', meetingValidation],
        ['recurrences', ['Create'], 'many', 'opt', scheduleRecurrenceValidation],
        ['runProjects', ['Connect'], 'many', 'opt', runProjectValidation],
        ['runRoutines', ['Connect'], 'many', 'opt', runRoutineValidation],
        ['translations', ['Create'], 'many', 'opt', scheduleTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        startTime: opt(startTime),
        endTime: opt(endTime),
        timezone: opt(timezone),
    }, [
        ['exceptions', ['Create', 'Update', 'Delete'], 'many', 'opt', scheduleExceptionValidation],
        ['focusModes', ['Connect', 'Disconnect'], 'many', 'opt', focusModeValidation],
        ['labels', ['Create', 'Connect', 'Disconnect'], 'many', 'opt', labelValidation],
        ['meetings', ['Connect', 'Disconnect'], 'many', 'opt', meetingValidation],
        ['recurrences', ['Create', 'Update', 'Delete'], 'many', 'opt', scheduleRecurrenceValidation],
        ['runProjects', ['Connect', 'Disconnect'], 'many', 'opt', runProjectValidation],
        ['runRoutines', ['Connect', 'Disconnect'], 'many', 'opt', runRoutineValidation],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', scheduleTranslationValidation],
    ], [], o),
}