import { ResourceListFor } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { resourceValidation } from "./resource.js";

const listForType = enumToYup(ResourceListFor);

export const resourceListTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: opt(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const resourceListValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        listForType: req(listForType),
    }, [
        ["listFor", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", resourceListTranslationValidation],
        ["resources", ["Create"], "many", "opt", resourceValidation, ["list"]],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceListTranslationValidation],
        ["resources", ["Create", "Update", "Delete"], "many", "opt", resourceValidation, ["list"]],
    ], [], d),
};
