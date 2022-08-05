import { useMutation } from "@apollo/client";
import { copy, copyVariables } from 'graphql/generated/copy';
import { fork, forkVariables } from 'graphql/generated/fork';
import { star, starVariables } from 'graphql/generated/star';
import { vote, voteVariables } from 'graphql/generated/vote';
import { copyMutation, forkMutation, starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { ReportFor, StarFor, VoteFor } from "@local/shared";
import { DeleteDialog, ListMenu, ReportDialog } from "..";
import { BaseObjectActionDialogProps, BaseObjectAction, ListMenuItemData } from "../types";
import {
    Cancel as CancelIcon,
    FileCopy as CopyIcon,
    DeleteForever as DeleteIcon,
    CardGiftcard as DonateIcon,
    ThumbDown as DownvoteIcon,
    Edit as EditIcon,
    ForkRight as ForkIcon,
    ReportProblem as ReportIcon,
    Share as ShareIcon,
    StarOutline as StarIcon,
    Star as UnstarIcon,
    ThumbUp as UpvoteIcon,
    Update as UpdateIcon,
    QueryStats as StatsIcon,
    SvgIconComponent,
} from "@mui/icons-material";
import { mutationWrapper } from "graphql/utils/mutationWrapper";
import { PubSub } from "utils";
import { CopyType, ForkType } from "graphql/generated/globalTypes";

/**
 * [label, Icon, iconColor, preview]
 */
const allOptionsMap: { [key in BaseObjectAction]: [string, SvgIconComponent, string, boolean] } = ({
    [BaseObjectAction.Copy]: ['Copy', CopyIcon, 'default', false],
    [BaseObjectAction.Delete]: ['Delete', DeleteIcon, "default", false],
    [BaseObjectAction.Donate]: ['Donate', DonateIcon, "default", true],
    [BaseObjectAction.Downvote]: ['Downvote', DownvoteIcon, "default", false],
    [BaseObjectAction.Edit]: ['Edit', EditIcon, "default", false],
    [BaseObjectAction.Fork]: ['Fork', ForkIcon, "default", false],
    [BaseObjectAction.Report]: ['Report', ReportIcon, "default", false],
    [BaseObjectAction.Share]: ['Share', ShareIcon, "default", false],
    [BaseObjectAction.Star]: ['Star', StarIcon, "#cbae30", false],
    [BaseObjectAction.Stats]: ['Stats', StatsIcon, "default", true],
    [BaseObjectAction.Unstar]: ['Unstar', UnstarIcon, "#cbae30", false],
    [BaseObjectAction.Update]: ['Update', UpdateIcon, "default", false],
    [BaseObjectAction.UpdateCancel]: ['Cancel Update', CancelIcon, "default", false],
    [BaseObjectAction.Upvote]: ['Upvote', UpvoteIcon, "default", false],
})

export const BaseObjectActionDialog = ({
    anchorEl,
    handleActionComplete,
    handleEdit,
    isStarred,
    isUpvoted,
    objectId,
    objectName,
    objectType,
    onClose,
    permissions,
    session,
    title,
    zIndex,
}: BaseObjectActionDialogProps) => {
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
                handleActionComplete(BaseObjectAction.Copy, data);
            },
        })
    }, [copy, handleActionComplete, objectId, objectName, objectType]);

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
                handleActionComplete(BaseObjectAction.Fork, data);
            }
        })
    }, [fork, handleActionComplete, objectId, objectName, objectType]);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        mutationWrapper({
            mutation: star,
            input: { isStar, starFor, forId: objectId },
            onSuccess: ({ data }) => {
                handleActionComplete(isStar ? BaseObjectAction.Star : BaseObjectAction.Unstar, data);
            }
        })
    }, [handleActionComplete, objectId, star]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, voteFor, forId: objectId },
            onSuccess: ({ data }) => {
                handleActionComplete(isUpvote ? BaseObjectAction.Upvote : BaseObjectAction.Downvote, data);
            }
        })
    }, [handleActionComplete, objectId, vote]);

    const onSelect = useCallback((action: BaseObjectAction) => {
        switch (action) {
            case BaseObjectAction.Copy:
                handleCopy();
                break;
            case BaseObjectAction.Delete:
                openDelete();
                break;
            case BaseObjectAction.Donate:
                openDonate();
                break;
            case BaseObjectAction.Downvote:
                handleVote(false, objectType as string as VoteFor);
                break;
            case BaseObjectAction.Edit:
                handleEdit();
                break;
            case BaseObjectAction.Fork:
                handleFork();
                break;
            case BaseObjectAction.Report:
                openReport();
                break;
            case BaseObjectAction.Share:
                openShare();
                break;
            case BaseObjectAction.Star:
                handleStar(true, objectType as string as StarFor);
                break;
            case BaseObjectAction.Unstar:
                handleStar(false, objectType as string as StarFor);
                break;
            case BaseObjectAction.Upvote:
                handleVote(true, objectType as string as VoteFor);
                break;
        }
        onClose();
    }, [handleCopy, handleEdit, handleFork, handleStar, handleVote, objectType, onClose, openDelete, openDonate, openReport, openShare]);

    /**
     * Actions that are available for the object, from top to bottom
     */
    const availableActions: BaseObjectAction[] = useMemo(() => {
        if (!permissions) return [];
        const isLoggedIn = session?.isLoggedIn === true;
        let options: BaseObjectAction[] = [];
        if (isLoggedIn && permissions.canVote) {
            options.push(isUpvoted ? BaseObjectAction.Downvote : BaseObjectAction.Upvote);
        }
        if (isLoggedIn && permissions.canStar) {
            options.push(isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
        }
        if (isLoggedIn && permissions.canFork) {
            options.push(BaseObjectAction.Copy);
            options.push(BaseObjectAction.Fork);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (isLoggedIn && permissions.canReport) {
            options.push(BaseObjectAction.Report);
        }
        if (isLoggedIn && permissions.canDelete) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [isStarred, isUpvoted, permissions, session?.isLoggedIn]);

    const data: ListMenuItemData<BaseObjectAction>[] = useMemo(() => {
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