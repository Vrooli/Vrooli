import { Reminder } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const reminderPartial: GqlPartial<Reminder> = {
    __typename: 'Reminder',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        name: true,
        description: true,
        dueDate: true,
        completed: true,
        index: true,
        reminderItems: () => relPartial(require('./reminderItem').reminderItemPartial, 'full', { omit: 'reminder' }),
        reminderList: () => relPartial(require('./reminderList').reminderListPartial, 'nav', { omit: 'reminders' }),
    },
}