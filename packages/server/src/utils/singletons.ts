import { i18nConfig } from "@vrooli/shared";
import i18next from "i18next";
import { ModelMap } from "../models/base/index.js";
import { AIServiceRegistry } from "../services/conversation/registry.js";
import { TokenEstimationRegistry } from "../services/conversation/tokens.js";
import { SocketService } from "../sockets/io.js";
import { initializeProfanity } from "./censor.js";

const debug = process.env.NODE_ENV === "development";

export async function initSingletons() {
    // Initialize translations
    await i18next.init(i18nConfig(debug));

    // Initialize singletons
    await SocketService.init();
    await ModelMap.init();
    await AIServiceRegistry.init();
    await TokenEstimationRegistry.init();

    // Initialize censor dictionary
    initializeProfanity();
}
