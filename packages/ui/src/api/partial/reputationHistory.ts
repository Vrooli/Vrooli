import { ReputationHistory } from "@shared/consts";
import { GqlPartial } from "types";

export const reputationHistoryPartial: GqlPartial<ReputationHistory> = {
    __typename: 'ReputationHistory',
    full: {
        id: true,
        created_at: true,
        updated_at: true,
        amount: true,
        event: true,
        objectId1: true,
        objectId2: true,
    },
}