import { transferPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const transferEndpoint = {
    findOne: toQuery('transfer', 'FindByIdInput', transferPartial, 'full'),
    findMany: toQuery('transfers', 'TransferSearchInput', ...toSearch(transferPartial)),
    requestSend: toMutation('transferRequestSend', 'TransferRequestSendInput', transferPartial, 'full'),
    requestReceive: toMutation('transferRequestReceive', 'TransferRequestReceiveInput', transferPartial, 'full'),
    transferUpdate: toMutation('transferUpdate', 'TransferUpdateInput', transferPartial, 'full'),
    transferCancel: toMutation('transferCancel', 'FindByIdInput', transferPartial, 'full'),
    transferAccept: toMutation('transferAccept', 'FindByIdInput', transferPartial, 'full'),
    transferDeny: toMutation('transferDeny', 'TransferDenyInput', transferPartial, 'full'),
}