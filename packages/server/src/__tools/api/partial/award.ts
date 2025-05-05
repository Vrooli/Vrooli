import { Award } from "@local/shared";
import { ApiPartial } from "../types.js";

export const award: ApiPartial<Award> = {
    common: {
        id: true,
        createdAt: true,
        updatedAt: true,
        tierCompletedAt: true,
        category: true,
        progress: true,
        title: true,
        description: true,
    },
};
