import { ReactionFor } from "@local/consts";
import { exists, getReactionScore } from "@local/utils";
import { useCallback } from "react";
import { mutationWrapper, useCustomMutation } from "../../api";
import { reactionReact } from "../../api/generated/endpoints/reaction_react";
import { ObjectActionComplete } from "../actions/objectActions";
import { PubSub } from "../pubsub";
export const useVoter = ({ objectId, objectType, onActionComplete, }) => {
    const [mutation] = useCustomMutation(reactionReact);
    const hasVotingSupport = exists(ReactionFor[objectType]);
    const handleVote = useCallback((emoji) => {
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasVotingSupport) {
            PubSub.get().publishSnack({ messageKey: "CopyNotSupported", severity: "Error" });
            return;
        }
        mutationWrapper({
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
//# sourceMappingURL=useVoter.js.map