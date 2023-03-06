import { TransferObjectType } from '@shared/consts';
import { enumToYup, id, message, opt, req, YupModel, yupObj } from "../utils";

const transferObjectType = enumToYup(TransferObjectType);

export const transferRequestSendValidation = ({ o }: any) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ['object', ['Connect'], 'one', 'req'],
    ['toOrganization', ['Connect'], 'one', 'opt'],
    ['toUser', ['Connect'], 'one', 'opt'],
], [['toOrganizationConnect', 'toUserConnect']], o);

export const transferRequestReceiveValidation = ({ o }: any) => yupObj({
    id: req(id),
    message: opt(message),
    objectType: req(transferObjectType),
}, [
    ['object', ['Connect'], 'one', 'req'],
    ['toOrganization', ['Connect'], 'one', 'opt'],
], [], o);

export const transferValidation: YupModel<false, true> = {
    // Cannot create a transfer through normal means. Must use request send/receive
    update: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], o),
}