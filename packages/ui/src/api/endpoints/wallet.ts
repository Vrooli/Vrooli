import { walletPartial } from '../partial';
import { toMutation, toQuery } from '../utils';

export const walletEndpoint = {
    findHandles: toQuery('findHandles', 'FindHandlesInput'),
    update: toMutation('walletUpdate', 'WalletUpdateInput', walletPartial, 'full'),
}