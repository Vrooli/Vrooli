import { Award } from "@local/shared";
import { GqlPartial } from "../types";

export const award: GqlPartial<Award> = {
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
};
