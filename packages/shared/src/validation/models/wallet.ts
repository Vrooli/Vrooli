import { id, name, opt, req, YupModel, yupObj } from "../utils";

export const walletValidation: YupModel<false, true> = {
    // Cannot create a wallet directly - must go through handshake
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], o),
};
