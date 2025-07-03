// AI_CHECK: TYPE_SAFETY=server-type-safety-fixes | LAST: 2025-06-29 - Fixed missing import path
import { i18nConfig } from "@vrooli/shared";
import i18next from "i18next";
import { ModelMap } from "../models/base/index.js";
import { AIServiceRegistry } from "../services/response/registry.js";
import { TokenEstimationRegistry } from "../services/response/tokens.js";
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
