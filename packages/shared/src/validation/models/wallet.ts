import { id, name, opt, req, YupModel, yupObj } from "../utils";

export const walletValidation: YupModel<["update"]> = {
    // Cannot create a wallet directly - must go through handshake
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], d),
};
