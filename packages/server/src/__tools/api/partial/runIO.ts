import { type RunIO } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const runIO: ApiPartial<RunIO> = {
    common: {
        id: true,
        data: true,
        nodeInputName: true,
        nodeName: true,
    },
};
