import { walletFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery } from 'graphql/utils';

export const walletEndpoint = {
    findHandles: toQuery('findHandles', 'FindHandlesInput', null),
    update: toMutation('walletUpdate', 'WalletUpdateInput', fullFields[1])
}