import { RoutineVersionTranslation, RoutineVersionYou } from "@shared/consts";
import { GqlPartial } from "types";

export const routineVersionTranslationPartial: GqlPartial<RoutineVersionTranslation> = {
    __typename: 'RoutineVersionTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
        instructions: true,
        name: true,
    }),
}

export const routineVersionYouPartial: GqlPartial<RoutineVersionYou> = {
    __typename: 'RoutineVersionYou',
    common: () => ({
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canRun: true,
        canView: true,
        canVote: true,
    }),
    full: () => ({
        runs: runRoutinePartial.full,
    }),
    list: () => ({
        runs: runRoutinePartial.list,
    })
}

export const listRoutineVersionFields = ['RoutineVersion', `{
    id
}`] as const;
export const routineVersionFields = ['RoutineVersion', `{
    id
}`] as const;