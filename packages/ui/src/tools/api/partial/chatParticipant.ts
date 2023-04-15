import { ChatParticipant } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const chatParticipant: GqlPartial<ChatParticipant> = {
    __typename: 'ChatParticipant',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
    },
    list: {
        user: async () => rel((await import('./user')).user, 'nav'),
    },
}