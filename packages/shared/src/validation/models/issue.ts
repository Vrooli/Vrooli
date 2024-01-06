import { IssueFor } from "@local/shared";
import { description, enumToYup, id, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";

const issueFor = enumToYup(IssueFor);

export const issueTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        name: req(name),
        description: opt(description),
    }),
    update: () => ({
        name: opt(name),
        description: opt(description),
    }),
});

export const issueValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        issueFor: req(issueFor),
    }, [
        ["for", ["Connect"], "one", "req"],
        ["labels", ["Connect", "Create"], "one", "opt", labelValidation],
        ["translations", ["Create"], "many", "opt", issueTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["labels", ["Connect", "Create", "Disconnect"], "one", "opt", labelValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", issueTranslationValidation],
    ], [], d),
};
