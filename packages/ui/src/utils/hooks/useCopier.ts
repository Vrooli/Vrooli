import { CopyInput, CopyResult, CopyType, exists } from "@local/shared";
import { mutationWrapper, useCustomMutation } from "api";
import { copyCopy } from "api/generated/endpoints/copy_copy";
import { useCallback } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { PubSub } from "utils/pubsub";

type UseCopierProps = {
    objectId: string | null | undefined;
    objectName: string | null | undefined;
    objectType: `${CopyType}`
    onActionComplete: (action: ObjectActionComplete.Fork, data: CopyResult) => void;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export const useCopier = ({
    objectId,
    objectName,
    objectType,
    onActionComplete
}: UseCopierProps) => {
    const [copy] = useCustomMutation<CopyResult, CopyInput>(copyCopy);

    const hasCopyingSupport = exists(CopyType[objectType]);

    const handleCopy = useCallback(() => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
            return;
        }
        if (!hasCopyingSupport) {
            PubSub.get().publishSnack({ messageKey: 'CopyNotSupported', severity: 'Error' });
            return;
        }
        mutationWrapper<CopyResult, CopyInput>({
            mutation: copy,
            input: { id: objectId, intendToPullRequest: true, objectType: CopyType[objectType] },
            successMessage: () => ({ key: 'CopySuccess', variables: { objectName: objectName ?? '' } }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data) },
        })
    }, [copy, hasCopyingSupport, objectId, objectName, objectType, onActionComplete]);

    return { handleCopy, hasCopyingSupport };
}