import * as yup from "yup";
import { ResourceUsedFor } from "../../api/types.js";
import { addHttps, enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, index, name } from "../utils/commonFields.js";
import { maxStrErr } from "../utils/errors.js";
import { handleRegex, urlRegex, urlRegexDev, walletAddressRegex } from "../utils/regex.js";
import { YupMutateParams, type YupModel } from "../utils/types.js";
import { resourceListValidation } from "./resourceList.js";

// Link must match one of the regex above
function link({ env = "production" }: { env?: YupMutateParams["env"] }) {
    return yup.string().trim().removeEmptyString().transform(addHttps).max(1024, maxStrErr).test(
        "link",
        "Must be a URL, Cardano payment address, or ADA Handle",
        (value: string | undefined) => {
            const regexForUrl = !env.startsWith("dev") ? urlRegex : urlRegexDev;
            return value !== undefined ? (regexForUrl.test(value) || walletAddressRegex.test(value) || handleRegex.test(value)) : true;
        },
    );
}
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
    ], [["listConnect", "listCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        index: opt(index),
        link: opt(link(d)),
        usedFor: opt(usedFor),
    }, [
        ["list", ["Connect", "Create"], "one", "opt", resourceListValidation, ["resources"]],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceTranslationValidation],
    ], [["listConnect", "listCreate", false]], d),
};
