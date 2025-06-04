import { type Count } from "@local/shared";
import { type ApiPartial } from "../types.js";

export const count: ApiPartial<Count> = {
    full: {
        count: true,
    },
};
