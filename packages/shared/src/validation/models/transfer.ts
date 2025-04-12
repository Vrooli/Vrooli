import { TransferObjectType } from "../../api/types.js";
import { YupMutateParams } from "../../validation/utils/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, message } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

const transferObjectType = enumToYup(TransferObjectType);

export function transferRequestSendValidation(d: YupMutateParams) {
    return yupObj({
        id: req(id),
        message: opt(message),
        objectType: req(transferObjectType),
    }, [
        ["object", ["Connect"], "one", "req"],
        ["toTeam", ["Connect"], "one", "opt"],
        ["toUser", ["Connect"], "one", "opt"],
    ], [["toTeamConnect", "toUserConnect", true]], d);
}

export function transferRequestReceiveValidation(d: YupMutateParams) {
    return yupObj({
        id: req(id),
        message: opt(message),
        objectType: req(transferObjectType),
    }, [
        ["object", ["Connect"], "one", "req"],
        ["toTeam", ["Connect"], "one", "opt"],
    ], [], d);
}

export const transferValidation: YupModel<["update"]> = {
    // Cannot create a transfer through normal means. Must use request send/receive
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};
