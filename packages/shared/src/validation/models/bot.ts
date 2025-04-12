import * as yup from "yup";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bio, bool, handle, id, imageFile, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { type YupModel } from "../utils/types.js";

export const botSettings = yup.string().trim().removeEmptyString().max(4096, maxStrErr);

export const botTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        bio: opt(bio),
    }),
    update: () => ({
        bio: opt(bio),
    }),
});

export const botValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        bannerImage: opt(imageFile),
        botSettings: req(botSettings),
        handle: opt(handle),
        isBotDepictingPerson: req(bool),
        isPrivate: opt(bool),
        name: req(name),
        profileImage: opt(imageFile),
    }, [
        ["translations", ["Create"], "many", "opt", botTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        botSettings: opt(botSettings),
        handle: opt(handle),
        isBotDepictingPerson: opt(bool),
        isPrivate: opt(bool),
        name: opt(name),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", botTranslationValidation],
    ], [], d),
};
