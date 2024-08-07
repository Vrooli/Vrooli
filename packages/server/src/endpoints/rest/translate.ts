import { translate_translate } from "../generated";
import { TranslateEndpoints } from "../logic/translate";
import { setupRoutes } from "./base";

export const TranslateRest = setupRoutes({
    "/translate": {
        get: [TranslateEndpoints.Query.translate, translate_translate],
    },
});
