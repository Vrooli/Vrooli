import { exists, setDotNotationValue } from "@shared/utils";
import { Dispatch, SetStateAction, useCallback } from "react";
import { SetLocation } from "types";
import { ObjectActionComplete } from "utils/actions";
import { getYouDot, ListObjectType } from "utils/display";
import { openObject } from "utils/navigation";
import { PubSub } from "utils/pubsub";

type UseObjectActionsProps = {
    object: ListObjectType | null | undefined;
    setLocation: SetLocation;
    setObject: Dispatch<SetStateAction<any>>;
}

/**
 * Hook for updating state and navigating upon completing an action
 */
export const useObjectActions = ({
    object,
    setLocation,
    setObject,
}: UseObjectActionsProps) => {
    const onActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        if (!exists(object)) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
            return;
        }
        switch (action) {
            case ObjectActionComplete.Bookmark:
            case ObjectActionComplete.BookmarkUndo:
                const isBookmarkedLocation = getYouDot(object, 'isBookmarked');
                const wasSuccessful = action === ObjectActionComplete.Bookmark ? data.success : exists(data);
                if (wasSuccessful && isBookmarkedLocation && object) setObject(setDotNotationValue(object, isBookmarkedLocation as any, wasSuccessful));
                break;
            case ObjectActionComplete.Fork:
                // Data is in first key with a value
                const forkData: any = Object.values(data).find((v) => typeof v === 'object');
                openObject(forkData, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                const isUpvotedLocation = getYouDot(object, 'isUpvoted');
                if (data.success && isUpvotedLocation && object) setObject(setDotNotationValue(object, isUpvotedLocation as any, action === ObjectActionComplete.VoteUp));
                break;
        }
    }, [object, setLocation, setObject]);

    return { onActionComplete };
}