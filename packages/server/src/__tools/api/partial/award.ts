import { Award } from "@local/shared";
import { ApiPartial } from "../types";

export const award: ApiPartial<Award> = {
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
