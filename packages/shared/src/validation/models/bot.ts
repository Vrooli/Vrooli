import * as yup from "yup";
import { bio, blankToUndefined, bool, handle, id, maxStrErr, name, opt, req, transRel, YupModel, yupObj } from "../utils";

export const botSettings = yup.string().trim().transform(blankToUndefined).max(4096, maxStrErr);

export const botTranslationValidation: YupModel = transRel({
    create: {
        bio: opt(bio),
    },
    update: {
        bio: opt(bio),
    },
});

export const botValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        botSettings: req(botSettings),
        handle: opt(handle),
        isPrivate: opt(bool),
        name: req(name),
    }, [
        ["translations", ["Create"], "many", "opt", botTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        botSettings: opt(botSettings),
        handle: opt(handle),
        isPrivate: opt(bool),
        name: opt(name),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", botTranslationValidation],
    ], [], o),
};
