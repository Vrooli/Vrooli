import { endpointPostReact, getReactionScore, GqlModelType, ReactInput, ReactionFor, Success } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { useCallback } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseVoterProps = {
    objectId: string | null | undefined;
    objectType: `${GqlModelType}`
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
    const [fetch] = useLazyFetch<ReactInput, Success>(endpointPostReact);

    const hasVotingSupport = objectType in ReactionFor;

    const handleVote = useCallback((emoji: string | null) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasVotingSupport) {
            PubSub.get().publishSnack({ messageKey: "VoteNotSupported", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ReactInput, Success>({
            fetch,
            inputs: {
                emoji,
                forConnect: objectId,
                reactionFor: ReactionFor[objectType],
            },
            onSuccess: (data) => { onActionComplete(getReactionScore(emoji) > 0 ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data); },
        });
    }, [hasVotingSupport, fetch, objectId, objectType, onActionComplete]);

    return { handleVote, hasVotingSupport };
};
