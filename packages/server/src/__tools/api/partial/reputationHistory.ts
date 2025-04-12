import { ReputationHistory } from "@local/shared";
import { ApiPartial } from "../types.js";

export const reputationHistory: ApiPartial<ReputationHistory> = {
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        amount: true,
        event: true,
        objectId1: true,
        objectId2: true,
    },
};
