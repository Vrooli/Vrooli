import { bool, description, id, name, opt, recurrEnd, recurrStart, req, timeZone, transRel, windowEnd, windowStart, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";

export const runProjectScheduleTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const runProjectScheduleValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        timeZone: opt(timeZone),
        windowStart: opt(windowStart),
        windowEnd: opt(windowEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['labels', ['Connect', 'Create'], 'many', 'opt', labelValidation],
        ['runProject', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', runProjectScheduleTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        timeZone: opt(timeZone),
        windowStart: opt(windowStart),
        windowEnd: opt(windowEnd),
        recurring: opt(bool),
        recurrStart: opt(recurrStart),
        recurrEnd: opt(recurrEnd),
    }, [
        ['labels', ['Connect', 'Create', 'Disconnect'], 'many', 'opt', labelValidation],
        ['runProject', ['Connect'], 'one', 'req'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', runProjectScheduleTranslationValidation],
    ], [], o),
}