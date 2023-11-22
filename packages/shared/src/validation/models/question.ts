import { maxStrErr, minStrErr, QuestionForType } from "@local/shared";
import * as yup from "yup";
import { bool, enumToYup, id, name, opt, referencing, req, transRel, YupModel, yupObj } from "../utils";

const forObjectType = enumToYup(QuestionForType);
const description = yup.string().trim().removeEmptyString().min(1, minStrErr).max(16384, maxStrErr);

export const questionTranslationValidation: YupModel = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const questionValidation: YupModel = {
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
