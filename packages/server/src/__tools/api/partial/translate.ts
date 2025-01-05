import { Translate } from "@local/shared";
import { GqlPartial } from "../types";

export const translate: GqlPartial<Translate> = {
    common: {
        fields: true,
        language: true,
    },
};
