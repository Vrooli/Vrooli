import { ResourceListFor } from "@local/shared";
import { description, enumToYup, id, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { resourceValidation } from "./resource";

const listForType = enumToYup(ResourceListFor);

export const resourceListTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: opt(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    },
});

export const resourceListValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        listForType: req(listForType),
    }, [
        ["listFor", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", resourceListTranslationValidation],
        ["resources", ["Create"], "many", "opt", resourceValidation, ["list"]],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
    }, [
        ["translations", ["Create", "Update", "Delete"], "many", "opt", resourceListTranslationValidation],
        ["resources", ["Create", "Update", "Delete"], "many", "opt", resourceValidation, ["list"]],
    ], [], o),
};
