import * as yup from "yup";
import { bio, bool, handle, id, imageFile, maxStrErr, name, opt, req, transRel, YupModel, yupObj } from "../utils";

export const botSettings = yup.string().trim().removeEmptyString().max(4096, maxStrErr);

export const botTranslationValidation: YupModel = transRel({
    create: () => ({
        bio: opt(bio),
    }),
    update: () => ({
        bio: opt(bio),
    }),
});

export const botValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        bannerImage: opt(imageFile),
        botSettings: req(botSettings),
        handle: opt(handle),
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
        isPrivate: opt(bool),
        name: opt(name),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", botTranslationValidation],
    ], [], d),
};
