import { Label, LabelYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const labelYou: GqlPartial<LabelYou> = {
    __typename: 'LabelYou',
    full: {
        canDelete: true,
        canUpdate: true,
    },
}

export const label: GqlPartial<Label> = {
    __typename: 'Label',
    common: {
        __define: {
            0: async () => rel((await import('./organization')).organization, 'nav'),
            1: async () => rel((await import('./user')).user, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        color: true,
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            }
        },
        you: () => rel(labelYou, 'full'),
    },
    full: {
        apisCount: true,
        issuesCount: true,
        meetingsCount: true,
        notesCount: true,
        projectsCount: true,
        routinesCount: true,
        runProjectSchedulesCount: true,
        runRoutineSchedulesCount: true,
        smartContractsCount: true,
        standardsCount: true,
    },
    list: {},
}