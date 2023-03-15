import { bool, description, id, intPositiveOrOne, name, opt, recurrEnd, recurrStart, req, timeZone, transRel, windowEnd, windowStart, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";

export const runRoutineScheduleTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const runRoutineScheduleValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        attemptAutomatic: opt(bool),
        maxAutomaticAttempts: opt(intPositiveOrOne),
        timeZone: opt(timeZone),
        windowStart: opt(windowStart),
        windowEnd: opt(windowEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['labels', ['Connect', 'Create'], 'many', 'opt', labelValidation],
        ['runRoutine', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', runRoutineScheduleTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        attemptAutomatic: opt(bool),
        maxAutomaticAttempts: opt(intPositiveOrOne),
        timeZone: opt(timeZone),
        windowStart: opt(windowStart),
        windowEnd: opt(windowEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation],
        ['runRoutine', ['Connect'], 'one', 'req'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', runRoutineScheduleTranslationValidation],
    ], [], o),
}