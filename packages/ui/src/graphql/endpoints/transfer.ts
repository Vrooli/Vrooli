import { transferFields as fullFields, listTransferFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const transferEndpoint = {
    findOne: toQuery('transfer', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('transfers', 'TransferSearchInput', toSearch(listFields)),
    requestSend: toMutation('transferRequestSend', 'TransferRequestSendInput', fullFields[1]),
    requestReceive: toMutation('transferRequestReceive', 'TransferRequestReceiveInput', fullFields[1]),
    transferUpdate: toMutation('transferUpdate', 'TransferUpdateInput', fullFields[1]),
    transferCancel: toMutation('transferCancel', 'FindByIdInput', fullFields[1]),
    transferAccept: toMutation('transferAccept', 'FindByIdInput', fullFields[1]),
    transferDeny: toMutation('transferDeny', 'TransferDenyInput', fullFields[1]),
}