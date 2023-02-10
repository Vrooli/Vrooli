import { TransferObjectType } from '@shared/consts';
import { enumToYup, id, message, opt, req, yupObj } from "../utils";

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

export const transferUpdateInput = ({ o }: any) => yupObj({
    id: req(id),
    message: opt(message),
}, [], [], o);