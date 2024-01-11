import { resourceListValidation, ResourceUsedFor, urlRegexDev, YupMutateParams } from "@local/shared";
import * as yup from "yup";
import { addHttps, description, enumToYup, handleRegex, id, index, maxStrErr, name, opt, req, transRel, urlRegex, walletAddressRegex, YupModel, yupObj } from "../utils";

// Link must match one of the regex above
const link = ({ env = "production" }: { env?: YupMutateParams["env"] }) =>
    yup.string().trim().removeEmptyString().transform(addHttps).max(1024, maxStrErr).test(
        "link",
        "Must be a URL, Cardano payment address, or ADA Handle",
        (value: string | undefined) => {
            const regexForUrl = !env.startsWith("dev") ? urlRegex : urlRegexDev;
            return value !== undefined ? (regexForUrl.test(value) || walletAddressRegex.test(value) || handleRegex.test(value)) : true;
        },
    );
const usedFor = enumToYup(ResourceUsedFor);

export const resourceTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: opt(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const resourceValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        index: opt(index),
        link: req(link(d)),
        usedFor: opt(usedFor),
    }, [
        ["list", ["Connect", "Create"], "one", "opt", resourceListValidation, ["resources"]],
        ["translations", ["Create"], "many", "opt", resourceTranslationValidation],
    ], [["listConnect", "listCreate"]], d),
    update: (d) => yupObj({
        id: req(id),
        index: opt(index),
        link: opt(link(d)),
        usedFor: opt(usedFor),
    }, [
        ["list", ["Connect", "Create"], "one", "opt", resourceListValidation, ["resources"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceTranslationValidation],
    ], [["listConnect", "listCreate"]], d),
};
