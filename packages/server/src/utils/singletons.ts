// AI_CHECK: TYPE_SAFETY=server-phase2-1 | LAST: 2025-07-03
import { i18nConfig } from "@vrooli/shared";
import i18next from "i18next";
import { initializeBcryptService } from "../auth/bcryptWrapper.js";
import { ModelMap } from "../models/base/index.js";
import { AIServiceRegistry } from "../services/response/registry.js";
import { TokenEstimationRegistry } from "../services/response/tokens.js";
import { SocketService } from "../sockets/io.js";
import { initializeProfanity } from "./censor.js";

const debug = process.env.NODE_ENV === "development";

export async function initSingletons(): Promise<void> {
    await i18next.init(i18nConfig(debug));
    await initializeBcryptService();
    await SocketService.init();
    await ModelMap.init();
    await AIServiceRegistry.init();
    await TokenEstimationRegistry.init();
    initializeProfanity();
}
