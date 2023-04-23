import { id, name, opt, req, yupObj } from "../utils";
export const walletValidation = {
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], o),
};
//# sourceMappingURL=wallet.js.map