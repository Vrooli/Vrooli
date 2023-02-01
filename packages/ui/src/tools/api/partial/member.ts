import { Member } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const member: GqlPartial<Member> = {
    __typename: 'Member',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isAdmin: true,
        permissions: true,
        organization: async () => rel((await import('./organization')).organization, 'nav'),
        user: async () => rel((await import('./user')).user, 'nav'),
    },
    full: {},
    list: {},
}