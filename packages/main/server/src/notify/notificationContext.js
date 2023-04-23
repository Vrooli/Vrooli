import { logger } from "../events";
export const parseSubscriptionContext = (contextJson) => {
    try {
        const settings = contextJson ? JSON.parse(contextJson) : {};
        return settings;
    }
    catch (error) {
        logger.error("Failed to parse notification subscription context", { trace: "0431" });
        return {};
    }
};
//# sourceMappingURL=notificationContext.js.map