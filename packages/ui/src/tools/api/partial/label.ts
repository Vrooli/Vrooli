import { Label, LabelYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const labelYouPartial: GqlPartial<LabelYou> = {
    __typename: 'LabelYou',
    full: {
        canDelete: true,
        canEdit: true,
    },
}

export const labelPartial: GqlPartial<Label> = {
    __typename: 'Label',
    common: {
        __define: {
            0: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
            1: async () => relPartial((await import('./user')).userPartial, 'nav'),
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
        you: () => relPartial(labelYouPartial, 'full'),
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