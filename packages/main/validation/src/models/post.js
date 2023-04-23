import { bool, id, opt, req, yupObj } from "../utils";
import { tagValidation } from "./tag";
import { resourceListValidation } from "./resourceList";
export const postValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        isPinned: opt(bool),
        isPrivate: opt(bool),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["repostedFrom", ["Connect"], "one", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
    ], [["organizationConnect", "userConnect"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isPinned: opt(bool),
        isPrivate: opt(bool),
    }, [
        ["resourceList", ["Update"], "one", "opt", resourceListValidation],
        ["tags", ["Connect", "Create", "Disconnect"], "many", "opt", tagValidation],
    ], [], o),
};
//# sourceMappingURL=post.js.map