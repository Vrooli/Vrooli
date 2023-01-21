import { Reminder } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { reminderItemPartial } from "./reminderItem";
import { reminderListPartial } from "./reminderList";

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
        reminderItems: () => relPartial(reminderItemPartial, 'full', { omit: 'reminder' }),
        reminderList: () => relPartial(reminderListPartial, 'nav', { omit: 'reminders' }),
    },
}