import { memberInviteFields as fullFields, listMemberInviteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const memberInviteEndpoint = {
    findOne: toQuery('memberInvite', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('memberInvites', 'MemberInviteSearchInput', toSearch(listFields)),
    create: toMutation('memberInviteCreate', 'MemberInviteCreateInput', fullFields[1]),
    update: toMutation('memberInviteUpdate', 'MemberInviteUpdateInput', fullFields[1]),
    accept: toMutation('memberInviteAccept', 'FindByIdInput', fullFields[1]),
    decline: toMutation('memberInviteDecline', 'FindByIdInput', fullFields[1])
}