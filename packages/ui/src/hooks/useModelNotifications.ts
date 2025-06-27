import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "../utils/pubsub.js";

export type ModelNotificationType = 
    | "model-unavailable"
    | "model-changed"
    | "model-restored"
    | "models-loading-failed";

export interface ModelNotificationData {
    type: ModelNotificationType;
    previousModel?: string;
    newModel?: string;
    reason?: string;
}

/**
 * Hook that listens for model-related notifications and displays them to the user.
 * Integrates with the app's notification system.
 */
export function useModelNotifications() {
    const { t } = useTranslation();
    const notificationTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
        // Subscribe to model notification events
        const unsubscribe = PubSub.get().subscribe("modelNotification", (data: ModelNotificationData) => {
            // Clear any existing timeout
            if (notificationTimeoutRef.current) {
                clearTimeout(notificationTimeoutRef.current);
                notificationTimeoutRef.current = null;
            }

            let message = "";
            let severity: "info" | "warning" | "error" | "success" = "info";

            switch (data.type) {
                case "model-unavailable":
                    message = t("ModelUnavailable", {
                        ns: "notification",
                        defaultValue: "Your preferred AI model is no longer available. Switched to {{newModel}}.",
                        newModel: data.newModel || "default model",
                    });
                    severity = "warning";
                    break;

                case "model-changed":
                    message = t("ModelChanged", {
                        ns: "notification",
                        defaultValue: "AI model changed to {{newModel}}.",
                        newModel: data.newModel || "default model",
                    });
                    severity = "info";
                    break;

                case "model-restored":
                    message = t("ModelRestored", {
                        ns: "notification",
                        defaultValue: "Your preferred AI model {{model}} is now available again.",
                        model: data.newModel || "model",
                    });
                    severity = "success";
                    break;

                case "models-loading-failed":
                    message = t("ModelsLoadingFailed", {
                        ns: "notification",
                        defaultValue: "Failed to load AI models. Using offline mode.",
                    });
                    severity = "error";
                    break;
            }

            if (message) {
                // Publish a snack notification
                PubSub.get().publish("snack", {
                    messageKey: message,
                    severity,
                    autoHideDuration: 6000,
                });
            }
        });

        return () => {
            unsubscribe();
            if (notificationTimeoutRef.current) {
                clearTimeout(notificationTimeoutRef.current);
            }
        };
    }, [t]);
}

/**
 * Publishes a model notification event.
 * Use this when model selection changes or fails.
 */
export function publishModelNotification(data: ModelNotificationData) {
    PubSub.get().publish("modelNotification", data);
}
