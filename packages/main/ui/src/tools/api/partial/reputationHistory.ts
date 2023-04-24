import { ReputationHistory } from "@local/shared";
import { GqlPartial } from "../types";

export const reputationHistory: GqlPartial<ReputationHistory> = {
    __typename: "ReputationHistory",
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
