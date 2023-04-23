import { CopyType } from "@local/consts";
import { exists } from "@local/utils";
import { useCallback } from "react";
import { mutationWrapper, useCustomMutation } from "../../api";
import { copyCopy } from "../../api/generated/endpoints/copy_copy";
import { ObjectActionComplete } from "../actions/objectActions";
import { PubSub } from "../pubsub";
export const useCopier = ({ objectId, objectName, objectType, onActionComplete, }) => {
    const [copy] = useCustomMutation(copyCopy);
    const hasCopyingSupport = exists(CopyType[objectType]);
    const handleCopy = useCallback(() => {
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasCopyingSupport) {
            PubSub.get().publishSnack({ messageKey: "CopyNotSupported", severity: "Error" });
            return;
        }
        mutationWrapper({
            mutation: copy,
            input: { id: objectId, intendToPullRequest: true, objectType: CopyType[objectType] },
            successMessage: () => ({ key: "CopySuccess", variables: { objectName: objectName ?? "" } }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data); },
        });
    }, [copy, hasCopyingSupport, objectId, objectName, objectType, onActionComplete]);
    return { handleCopy, hasCopyingSupport };
};
//# sourceMappingURL=useCopier.js.map