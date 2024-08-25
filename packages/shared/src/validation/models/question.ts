import * as yup from "yup";
import { QuestionForType } from "../../api/generated/graphqlTypes";
import { YupModel, bool, enumToYup, id, opt, referencing, req, transRel, yupObj } from "../utils";
import { maxStrErr, minStrErr } from "../utils/errors";

const forObjectType = enumToYup(QuestionForType);
const name = yup.string().trim().removeEmptyString().min(1, minStrErr).max(128, maxStrErr);
const description = yup.string().trim().removeEmptyString().min(1, minStrErr).max(16384, maxStrErr);

export const questionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const questionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
        referencing: opt(referencing),
        forObjectType: opt(forObjectType),
    }, [
        ["forObject", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", questionTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["acceptedAnswer", ["Connect"], "one", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", questionTranslationValidation],
    ], [], d),
};
