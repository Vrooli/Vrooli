import { ReactInput, ReactionFor, Success } from ":local/consts";
import { exists, getReactionScore } from ":local/utils";
import { useCallback } from "react";
import { mutationWrapper, useCustomMutation } from "../../api";
import { reactionReact } from "../../api/generated/endpoints/reaction_react";
import { ObjectActionComplete } from "../actions/objectActions";
import { PubSub } from "../pubsub";

type UseVoterProps = {
    objectId: string | null | undefined;
    objectType: `${ReactionFor}`
    onActionComplete: (action: ObjectActionComplete.VoteDown | ObjectActionComplete.VoteUp, data: Success) => void;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export const useVoter = ({
    objectId,
    objectType,
    onActionComplete,
}: UseVoterProps) => {
    const [mutation] = useCustomMutation<Success, ReactInput>(reactionReact);

    const hasVotingSupport = exists(ReactionFor[objectType]);

    const handleVote = useCallback((emoji: string | null) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasVotingSupport) {
            PubSub.get().publishSnack({ messageKey: "CopyNotSupported", severity: "Error" });
            return;
        }
        mutationWrapper<Success, ReactInput>({
            mutation,
            input: {
                emoji,
                forConnect: objectId,
                reactionFor: ReactionFor[objectType],
            },
            onSuccess: (data) => { onActionComplete(getReactionScore(emoji) > 0 ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data); },
        });
    }, [hasVotingSupport, mutation, objectId, objectType, onActionComplete]);

    return { handleVote, hasVotingSupport };
};
