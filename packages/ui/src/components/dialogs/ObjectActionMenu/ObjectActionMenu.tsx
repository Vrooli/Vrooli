import { useMutation } from "@apollo/client";
import { copy, copyVariables } from 'graphql/generated/copy';
import { fork, forkVariables } from 'graphql/generated/fork';
import { star, starVariables } from 'graphql/generated/star';
import { vote, voteVariables } from 'graphql/generated/vote';
import { copyMutation, forkMutation, starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { ReportFor, StarFor, VoteFor } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog } from "..";
import { ObjectActionMenuProps, ListMenuItemData, ObjectActionComplete, ObjectAction } from "../types";
import { mutationWrapper } from "graphql/utils/graphqlWrapper";
import { PubSub } from "utils";
import { CopyType, ForkType } from "graphql/generated/globalTypes";
import { BranchIcon, CopyIcon, DeleteIcon, DonateIcon, DownvoteWideIcon, EditIcon, ReportIcon, SearchIcon, ShareIcon, StarFilledIcon, StarOutlineIcon, StatsIcon, SvgComponent, UpvoteWideIcon } from "@shared/icons";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";

/**
 * [label, Icon, iconColor, preview]
 */
const allOptionsMap: { [key in ObjectAction]: [string, SvgComponent, string, boolean] } = ({
    [ObjectAction.Copy]: ['Copy', CopyIcon, 'default', false],
    [ObjectAction.Delete]: ['Delete', DeleteIcon, "default", false],
    [ObjectAction.Donate]: ['Donate', DonateIcon, "default", true],
    [ObjectAction.Edit]: ['Edit', EditIcon, "default", false],
    [ObjectAction.FindInPage]: ['Find...', SearchIcon, "default", false],
    [ObjectAction.Fork]: ['Fork', BranchIcon, "default", false],
    [ObjectAction.Report]: ['Report', ReportIcon, "default", false],
    [ObjectAction.Share]: ['Share', ShareIcon, "default", false],
    [ObjectAction.Star]: ['Star', StarFilledIcon, "#cbae30", false],
    [ObjectAction.StarUndo]: ['Unstar', StarOutlineIcon, "#cbae30", false],
    [ObjectAction.Stats]: ['Stats', StatsIcon, "default", true],
    [ObjectAction.VoteDown]: ['Downvote', DownvoteWideIcon, "default", false],
    [ObjectAction.VoteUp]: ['Upvote', UpvoteWideIcon, "default", false],
})

export const ObjectActionMenu = ({
    anchorEl,
    isStarred,
    isUpvoted,
    objectId,
    objectName,
    objectType,
    onActionComplete,
    onActionStart,
    onClose,
    permissions,
    session,
    title,
    zIndex,
}: ObjectActionMenuProps) => {
    // States
    const [deleteOpen, setDeleteOpen] = useState<boolean>(false);
    const [donateOpen, setDonateOpen] = useState<boolean>(false);
    const [shareOpen, setShareOpen] = useState<boolean>(false);
    const [reportOpen, setReportOpen] = useState<boolean>(false);

    const openDelete = useCallback(() => setDeleteOpen(true), [setDeleteOpen]);
    const closeDelete = useCallback(() => setDeleteOpen(false), [setDeleteOpen]);

    const openDonate = useCallback(() => setDonateOpen(true), [setDonateOpen]);
    const closeDonate = useCallback(() => setDonateOpen(false), [setDonateOpen]);

    const openShare = useCallback(() => setShareOpen(true), [setShareOpen]);
    const closeShare = useCallback(() => setShareOpen(false), [setShareOpen]);

    const openReport = useCallback(() => setReportOpen(true), [setReportOpen]);
    const closeReport = useCallback(() => setReportOpen(false), [setReportOpen]);

    // Mutations
    const [copy] = useMutation<copy, copyVariables>(copyMutation);
    const [fork] = useMutation<fork, forkVariables>(forkMutation);
    const [star] = useMutation<star, starVariables>(starMutation);
    const [vote] = useMutation<vote, voteVariables>(voteMutation);

    const handleCopy = useCallback(() => {
        // Check if objectType can be converted to CopyType
        const copyType = CopyType[objectType];
        if (!copyType) {
            PubSub.get().publishSnack({ message: 'Copy not supported on this object type.', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: copy,
            input: { id: objectId, objectType: copyType },
            onSuccess: ({ data }) => {
                PubSub.get().publishSnack({ message: `${objectName} copied.`, severity: 'success' });
                onActionComplete(ObjectActionComplete.Copy, data);
            },
        })
    }, [copy, objectId, objectName, objectType, onActionComplete]);

    const handleFork = useCallback(() => {
        // Check if objectType can be converted to ForkType
        const forkType = ForkType[objectType];
        if (!forkType) {
            PubSub.get().publishSnack({ message: 'Fork not supported on this object type.', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: fork,
            input: { id: objectId, objectType: forkType },
            onSuccess: ({ data }) => {
                PubSub.get().publishSnack({ message: `${objectName} forked.`, severity: 'success' });
                onActionComplete(ObjectActionComplete.Fork, data);
            }
        })
    }, [fork, objectId, objectName, objectType, onActionComplete]);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        mutationWrapper({
            mutation: star,
            input: { isStar, starFor, forId: objectId },
            onSuccess: ({ data }) => {
                onActionComplete(isStar ? ObjectActionComplete.Star : ObjectActionComplete.StarUndo, data);
            }
        })
    }, [objectId, onActionComplete, star]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, voteFor, forId: objectId },
            onSuccess: ({ data }) => {
                onActionComplete(isUpvote ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data);
            }
        })
    }, [objectId, onActionComplete, vote]);

    const onSelect = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Copy:
                handleCopy();
                break;
            case ObjectAction.Delete:
                openDelete();
                break;
            case ObjectAction.Donate:
                openDonate();
                break;
            case ObjectAction.Edit:
                onActionStart(ObjectAction.Edit);
                break;
            case ObjectAction.FindInPage:
                PubSub.get().publishFindInPage();
                break;
            case ObjectAction.Fork:
                handleFork();
                break;
            case ObjectAction.Report:
                openReport();
                break;
            case ObjectAction.Share:
                openShare();
                break;
            case ObjectAction.Star:
            case ObjectAction.StarUndo:
                handleStar(action === ObjectAction.Star, objectType as string as StarFor);
                break;
            case ObjectAction.Stats:
                onActionStart(ObjectAction.Stats);
                break;
            case ObjectAction.VoteDown:
            case ObjectAction.VoteUp:
                handleVote(action === ObjectAction.VoteUp, objectType as string as VoteFor);
        }
        onClose();
    }, [handleCopy, handleFork, handleStar, handleVote, objectType, onActionStart, onClose, openDelete, openDonate, openReport, openShare]);

    /**
     * Actions that are available for the object, from top to bottom
     */
    const availableActions: ObjectAction[] = useMemo(() => {
        if (!permissions) return [];
        const isLoggedIn = session?.isLoggedIn === true;
        let options: ObjectAction[] = [];
        if (isLoggedIn && permissions.canVote) {
            options.push(isUpvoted ? ObjectAction.VoteDown : ObjectAction.VoteUp);
        }
        if (isLoggedIn && permissions.canStar) {
            options.push(isStarred ? ObjectAction.StarUndo : ObjectAction.Star);
        }
        if (isLoggedIn && permissions.canFork) {
            options.push(ObjectAction.Copy);
            options.push(ObjectAction.Fork);
        }
        options.push(ObjectAction.Stats, ObjectAction.Donate, ObjectAction.Share, ObjectAction.FindInPage);
        if (isLoggedIn && permissions.canReport) {
            options.push(ObjectAction.Report);
        }
        if (isLoggedIn && permissions.canDelete) {
            options.push(ObjectAction.Delete);
        }
        return options;
    }, [isStarred, isUpvoted, permissions, session?.isLoggedIn]);

    const data: ListMenuItemData<ObjectAction>[] = useMemo(() => {
        // Convert options to ListMenuItemData
        return availableActions
            .map(option => {
                const [label, Icon, iconColor, preview] = allOptionsMap[option];
                return {
                    label,
                    value: option,
                    Icon,
                    iconColor,
                    preview,
                }
            })
    }, [availableActions]);

    return (
        <>
            {/* Delete routine confirmation dialog */}
            <DeleteDialog
                isOpen={deleteOpen}
                objectId={objectId}
                objectType={objectType as any}
                objectName={objectName}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />
            {/* Report dialog */}
            <ReportDialog
                forId={objectId}
                onClose={closeReport}
                open={reportOpen}
                reportFor={objectType as string as ReportFor}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Share dialog */}
            <ShareObjectDialog
                objectType={objectType}
                open={shareOpen}
                onClose={closeShare}
                zIndex={zIndex + 1}
            />
            {/* Actual action dialog */}
            <ListMenu
                anchorEl={anchorEl}
                data={data}
                id={`${objectType}-options-menu-${objectId}`}
                onClose={onClose}
                onSelect={onSelect}
                title={title}
                zIndex={zIndex}
            />
        </>
    )
}