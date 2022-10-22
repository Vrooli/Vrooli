import { useMutation } from "@apollo/client";
import { copyVariables, copy_copy } from 'graphql/generated/copy';
import { forkVariables, fork_fork } from 'graphql/generated/fork';
import { starVariables, star_star } from 'graphql/generated/star';
import { voteVariables, vote_vote } from 'graphql/generated/vote';
import { copyMutation, forkMutation, starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { ReportFor, StarFor, VoteFor } from "@shared/consts";
import { DeleteDialog, ListMenu, ReportDialog, SnackSeverity } from "..";
import { ObjectActionMenuProps, ListMenuItemData, ObjectActionComplete, ObjectAction } from "../types";
import { mutationWrapper } from "graphql/utils/graphqlWrapper";
import { getListItemIsStarred, getListItemIsUpvoted, getListItemPermissions, getListItemTitle, getUserLanguages, ObjectType, PubSub } from "utils";
import { CopyType, DeleteOneType, ForkType } from "graphql/generated/globalTypes";
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
    [ObjectAction.Star]: ['Star', StarOutlineIcon, "#cbae30", false],
    [ObjectAction.StarUndo]: ['Unstar', StarFilledIcon, "#cbae30", false],
    [ObjectAction.Stats]: ['Stats', StatsIcon, "default", true],
    [ObjectAction.VoteDown]: ['Downvote', DownvoteWideIcon, "default", false],
    [ObjectAction.VoteUp]: ['Upvote', UpvoteWideIcon, "default", false],
})

export const ObjectActionMenu = ({
    anchorEl,
    object,
    onActionComplete,
    onActionStart,
    onClose,
    session,
    title,
    zIndex,
}: ObjectActionMenuProps) => {

    const { id, isStarred, isUpvoted, name, objectType, permissions } = useMemo(() => ({
        id: object?.id,
        isStarred: getListItemIsStarred(object),
        isUpvoted: getListItemIsUpvoted(object),
        name: getListItemTitle(object, getUserLanguages(session)),
        objectType: object?.__typename as ObjectType,
        permissions: getListItemPermissions(object, session),
    }), [object, session]);

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
    const [copy] = useMutation(copyMutation);
    const [fork] = useMutation(forkMutation);
    const [star] = useMutation(starMutation);
    const [vote] = useMutation(voteMutation);

    const handleCopy = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to CopyType
        const copyType = CopyType[objectType];
        if (!copyType) {
            PubSub.get().publishSnack({ message: 'Copy not supported on this object type.', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<copy_copy, copyVariables>({
            mutation: copy,
            input: { id, objectType: copyType },
            successMessage: () => `${name} copied.`,
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Copy, data) },
        })
    }, [copy, id, name, objectType, onActionComplete]);

    const handleFork = useCallback(() => {
        if (!id) return;
        // Check if objectType can be converted to ForkType
        const forkType = ForkType[objectType];
        if (!forkType) {
            PubSub.get().publishSnack({ message: 'Fork not supported on this object type.', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<fork_fork, forkVariables>({
            mutation: fork,
            input: { id, objectType: forkType },
            successMessage: () => `${name} forked.`,
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Fork, data) },
        })
    }, [fork, id, name, objectType, onActionComplete]);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        if (!id) return;
        mutationWrapper<star_star, starVariables>({
            mutation: star,
            input: { isStar, starFor, forId: id },
            onSuccess: (data) => { onActionComplete(isStar ? ObjectActionComplete.Star : ObjectActionComplete.StarUndo, data) },
        })
    }, [id, onActionComplete, star]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        if (!id) return;
        mutationWrapper<vote_vote, voteVariables>({
            mutation: vote,
            input: { isUpvote, voteFor, forId: id },
            onSuccess: (data) => { onActionComplete(isUpvote ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data) },
        })
    }, [id, onActionComplete, vote]);

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
            {id && objectType in DeleteOneType && <DeleteDialog
                isOpen={deleteOpen}
                objectId={id}
                objectType={objectType as any}
                objectName={name}
                handleClose={closeDelete}
                zIndex={zIndex + 1}
            />}
            {/* Report dialog */}
            {id && objectType in ReportFor && <ReportDialog
                forId={id}
                onClose={closeReport}
                open={reportOpen}
                reportFor={objectType as any}
                session={session}
                zIndex={zIndex + 1}
            />}
            {/* Share dialog */}
            <ShareObjectDialog
                object={object}
                open={shareOpen}
                onClose={closeShare}
                zIndex={zIndex + 1}
            />
            {/* Actual action dialog */}
            <ListMenu
                anchorEl={anchorEl}
                data={data}
                id={`${objectType}-options-menu-${id}`}
                onClose={onClose}
                onSelect={onSelect}
                title={title}
                zIndex={zIndex}
            />
        </>
    )
}