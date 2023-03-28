import { Reminder, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { ReminderShape } from 'utils/shape/models/reminder';

export * from './ReminderCreate/ReminderCreate';
export * from './ReminderUpdate/ReminderUpdate';
export * from './ReminderView/ReminderView';

export const reminderInitialValues = (
    session: Session | undefined,
    reminderListId: string | undefined,
    existing?: Reminder | null | undefined
): ReminderShape => ({
    __typename: 'Reminder' as const,
    id: DUMMY_ID,
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    name: '',
    reminderList: {
        __typename: 'ReminderList' as const,
        id: reminderListId ?? DUMMY_ID,
    },
    reminderItems: [],
    ...existing,
});