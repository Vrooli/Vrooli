import { Count } from "@local/shared";
import { ApiPartial } from "../types.js";

export const count: ApiPartial<Count> = {
    full: {
        count: true,
    },
};
