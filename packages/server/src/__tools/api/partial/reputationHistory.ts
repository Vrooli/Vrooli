import { type ReputationHistory } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const reputationHistory: ApiPartial<ReputationHistory> = {
    full: {
        id: true,
        createdAt: true,
        updatedAt: true,
        amount: true,
        event: true,
        objectId1: true,
        objectId2: true,
    },
};
