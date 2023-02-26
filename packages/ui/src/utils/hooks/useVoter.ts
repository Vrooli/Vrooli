import { Success, VoteFor, VoteInput } from "@shared/consts";
import { exists } from "@shared/utils";
import { mutationWrapper, useCustomMutation } from "api";
import { voteVote } from "api/generated/endpoints/vote";
import { useCallback } from "react";
import { ObjectActionComplete } from "utils/actions";
import { PubSub } from "utils/pubsub";

type UseVoterProps = {
    objectId: string | null | undefined;
    objectType: `${VoteFor}`
    onActionComplete: (action: ObjectActionComplete.VoteDown | ObjectActionComplete.VoteUp, data: Success) => void;
}

/**
 * Hook for simplifying the use of voting on an object
 */
export const useVoter = ({
    objectId,
    objectType,
    onActionComplete
}: UseVoterProps) => {
    const [vote] = useCustomMutation<Success, VoteInput>(voteVote);

    const hasVotingSupport = exists(VoteFor[objectType]);

    const handleVote = useCallback((isUpvote: boolean | null) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: `CouldNotRead${objectType}`, severity: 'Error' });
            return;
        }
        if(!hasVotingSupport) {
            PubSub.get().publishSnack({ messageKey: 'CopyNotSupported', severity: 'Error' });
            return;
        }
        mutationWrapper<Success, VoteInput>({
            mutation: vote,
            input: { isUpvote, forConnect: objectId, voteFor: VoteFor[objectType] },
            onSuccess: (data) => { onActionComplete(isUpvote ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data) },
        })
    }, [hasVotingSupport, objectId, objectType, onActionComplete, vote]);

    return { handleVote, hasVotingSupport };
}