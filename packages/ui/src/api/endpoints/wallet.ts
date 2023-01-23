import { walletPartial } from 'api/partial';
import { toMutation, toQuery } from 'api/utils';

export const walletEndpoint = {
    findHandles: toQuery('findHandles', 'FindHandlesInput'),
    update: toMutation('walletUpdate', 'WalletUpdateInput', walletPartial, 'full'),
}