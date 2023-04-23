import { bool, id, opt, permissions, req, yupObj } from "../utils";
export const memberValidation = {
    update: ({ o }) => yupObj({
        id: req(id),
        isAdmin: opt(bool),
        permissions: opt(permissions),
    }, [], [], o),
};
//# sourceMappingURL=member.js.map