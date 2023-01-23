import { Member } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const memberPartial: GqlPartial<Member> = {
    __typename: 'Member',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isAdmin: true,
        permissions: true,
        organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
        user: () => relPartial(require('./user').userPartial, 'nav'),
    }
}