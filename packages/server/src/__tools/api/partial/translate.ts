import { Translate } from "@local/shared";
import { ApiPartial } from "../types";

export const translate: ApiPartial<Translate> = {
    common: {
        fields: true,
        language: true,
    },
};
