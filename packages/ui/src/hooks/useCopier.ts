import { CopyInput, CopyResult, CopyType, endpointPostCopy, GqlModelType } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { useCallback } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseCopierProps = {
    objectId: string | null | undefined;
    objectName: string | null | undefined;
    objectType: `${GqlModelType}`
    onActionComplete: (action: ObjectActionComplete.Fork, data: CopyResult) => unknown;
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

    const hasCopyingSupport = objectType in CopyType;

    const handleCopy = useCallback(() => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasCopyingSupport) {
            PubSub.get().publishSnack({ messageKey: "CopyNotSupported", severity: "Error" });
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
