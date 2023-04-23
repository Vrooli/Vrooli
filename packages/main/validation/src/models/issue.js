import { IssueFor } from "@local/consts";
import { description, enumToYup, id, name, opt, req, transRel, yupObj } from "../utils";
import { labelValidation } from "./label";
const issueFor = enumToYup(IssueFor);
export const issueTranslationValidation = transRel({
    create: {
        name: req(name),
        description: opt(description),
    },
    update: {
        name: opt(name),
        description: opt(description),
    },
});
export const issueValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        issueFor: req(issueFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["labels", ["Connect", "Create"], "one", "opt", labelValidation],
        ["translations", ["Create"], "many", "opt", issueTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["labels", ["Connect", "Create", "Disconnect"], "one", "opt", labelValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", issueTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=issue.js.map