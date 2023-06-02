import { translate_translate } from "@local/shared";
import { TranslateEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const TranslateRest = setupRoutes({
    "/translate": {
        get: [TranslateEndpoints.Query.translate, translate_translate],
    },
});
