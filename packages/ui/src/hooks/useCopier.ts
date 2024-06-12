import { CopyInput, CopyResult, CopyType, endpointPostCopy, GqlModelType } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { useCallback } from "react";
import { ActionCompletePayloads, ObjectActionComplete } from "utils/actions/objectActions";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseCopierProps = {
    objectId: string | null | undefined;
    objectName: string | null | undefined;
    objectType: `${GqlModelType}` | undefined;
    onActionComplete: <T extends "Fork">(action: T, data: ActionCompletePayloads[T]) => unknown;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export const useCopier = ({
    objectId,
    objectName,
    objectType,
    onActionComplete,
}: UseCopierProps) => {
    const [copy] = useLazyFetch<CopyInput, CopyResult>(endpointPostCopy);

    const hasCopyingSupport = objectType && objectType in CopyType;

    const handleCopy = useCallback(() => {
        // Validate objectId and objectType
        if (!objectType || !objectId) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasCopyingSupport) {
            console.error("Cannot copy this object type", objectType, objectId);
            PubSub.get().publish("snack", { messageKey: "CopyNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<CopyInput, CopyResult>({
            fetch: copy,
            inputs: { id: objectId, intendToPullRequest: true, objectType: CopyType[objectType] },
            successMessage: () => ({ messageKey: "CopySuccess", messageVariables: { objectName: objectName ?? "" } }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data); },
        });
    }, [copy, hasCopyingSupport, objectId, objectName, objectType, onActionComplete]);

    return { handleCopy, hasCopyingSupport };
};
