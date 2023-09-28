import { resourceListValidation, ResourceUsedFor } from "@local/shared";
import * as yup from "yup";
import { addHttps, blankToUndefined, description, enumToYup, handleRegex, id, index, maxStrErr, name, opt, req, transRel, urlRegex, walletAddressRegex, YupModel, yupObj } from "../utils";

// Link must match one of the regex above
const link = yup.string().trim().transform(blankToUndefined).transform(addHttps).max(1024, maxStrErr).test(
    "link",
    "Must be a URL, Cardano payment address, or ADA Handle",
    (value: string | undefined) => {
        return value !== undefined ? (urlRegex.test(value) || walletAddressRegex.test(value) || handleRegex.test(value)) : true;
    },
);
const usedFor = enumToYup(ResourceUsedFor);

export const resourceTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: opt(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});

export const resourceValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        link: req(link),
        usedFor: opt(usedFor),
    }, [
        ["list", ["Connect", "Create"], "one", "req", resourceListValidation, ["resources"]],
        ["translations", ["Create"], "many", "opt", resourceTranslationValidation],
    ], [["listConnect", "listCreate"]], o),
    update: ({ o }) => yupObj({
        id: req(id),
        index: opt(index),
        link: opt(link),
        usedFor: opt(usedFor),
    }, [
        ["list", ["Connect", "Create"], "one", "opt", resourceListValidation, ["resources"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceTranslationValidation],
    ], [["listConnect", "listCreate"]], o),
};
