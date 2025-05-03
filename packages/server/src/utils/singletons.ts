import { i18nConfig } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from "../models/base/index.js";
import { SocketService } from "../sockets/io.js";
import { LlmServiceRegistry } from "../tasks/llm/registry.js";
import { TokenEstimationRegistry } from "../tasks/llm/tokenEstimator.js";
import { initializeProfanity } from "./censor.js";

const debug = process.env.NODE_ENV === "development";

export async function initSingletons() {
    // Initialize translations
    await i18next.init(i18nConfig(debug));

    // Initialize singletons
    await SocketService.init();
    await ModelMap.init();
    await LlmServiceRegistry.init();
    await TokenEstimationRegistry.init();

    // Initialize censor dictionary
    initializeProfanity();
}
