import { IssueFor } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

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
        ["translations", ["Create"], "many", "opt", issueTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", issueTranslationValidation],
    ], [], d),
};
