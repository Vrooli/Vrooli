import { RunIO } from "@local/shared";
import { ApiPartial } from "../types.js";

export const runIO: ApiPartial<RunIO> = {
    common: {
        id: true,
        data: true,
        nodeInputName: true,
        nodeName: true,
    },
};
