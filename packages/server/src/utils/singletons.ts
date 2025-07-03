// AI_CHECK: TYPE_SAFETY=server-phase2-1 | LAST: 2025-07-03
import { i18nConfig } from "@vrooli/shared";
import i18next from "i18next";
import { ModelMap } from "../models/base/index.js";
import { AIServiceRegistry } from "../services/response/registry.js";
import { TokenEstimationRegistry } from "../services/response/tokens.js";
import { SocketService } from "../sockets/io.js";
import { initializeProfanity } from "./censor.js";

const debug = process.env.NODE_ENV === "development";

export async function initSingletons(): Promise<void> {
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
