import { transferFields as fullFields, listTransferFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const transferEndpoint = {
    findOne: toQuery('transfer', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('transfers', 'TransferSearchInput', [listFields], toSearch(listFields)),
    requestSend: toMutation('transferRequestSend', 'TransferRequestSendInput', [fullFields], `...fullFields`),
    requestReceive: toMutation('transferRequestReceive', 'TransferRequestReceiveInput', [fullFields], `...fullFields`),
    transferUpdate: toMutation('transferUpdate', 'TransferUpdateInput', [fullFields], `...fullFields`),
    transferCancel: toMutation('transferCancel', 'FindByIdInput', [fullFields], `...fullFields`),
    transferAccept: toMutation('transferAccept', 'FindByIdInput', [fullFields], `...fullFields`),
    transferDeny: toMutation('transferDeny', 'TransferDenyInput', [fullFields], `...fullFields`),
}