import { walletPartial } from 'graphql/partial';
import { toMutation, toQuery } from 'graphql/utils';

export const walletEndpoint = {
    findHandles: toQuery('findHandles', 'FindHandlesInput'),
    update: toMutation('walletUpdate', 'WalletUpdateInput', walletPartial, 'full'),
}