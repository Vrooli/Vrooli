import { Award } from "@shared/consts";
import { GqlPartial } from "../types";

export const award: GqlPartial<Award> = {
    __typename: 'Award',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        timeCurrentTierCompleted: true,
        category: true,
        progress: true,
        title: true,
        description: true,
    },
    full: {},
    list: {},
}