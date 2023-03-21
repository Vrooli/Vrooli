import { FocusMode } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const focusMode: GqlPartial<FocusMode> = {
    __typename: 'FocusMode',
    common: {
        id: true,
        name: true,
        description: true,
    },
    full: {
        __define: {
            0: async () => rel((await import('./label')).label, 'full'),
            1: async () => rel((await import('./schedule')).schedule, 'common'),
        },
        filters: async () => rel((await import('./focusModeFilter')).focusModeFilter, 'full'),
        labels: { __use: 0 },
        reminderList: async () => rel((await import('./reminderList')).reminderList, 'full', { omit: 'focusMode' }),
        schedule: { __use: 1 },
    },
    list: {
        __define: {
            0: async () => rel((await import('./label')).label, 'list'),
            1: async () => rel((await import('./schedule')).schedule, 'common'),
        },
        labels: { __use: 0 },
        schedule: { __use: 1 },
    }
}