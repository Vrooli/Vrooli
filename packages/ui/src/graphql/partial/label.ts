import { Label, LabelYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { organizationPartial } from "./organization";
import { userPartial } from "./user";

export const labelYouPartial: GqlPartial<LabelYou> = {
    __typename: 'LabelYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
    }),
}

export const labelPartial: GqlPartial<Label> = {
    __typename: 'Label',
    common: () => ({
        id: true,
        created_at: true,
        updated_at: true,
        color: true,
        owner: {
            __union: {
                Organization: relPartial(organizationPartial, 'nav'),
                User: relPartial(userPartial, 'nav'),
            }
        },
        you: relPartial(labelYouPartial, 'full'),
    }),
    full: () => ({
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
    }),
}