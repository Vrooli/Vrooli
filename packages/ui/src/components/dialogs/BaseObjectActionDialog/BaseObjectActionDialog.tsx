import { useMutation } from "@apollo/client";
import { star } from 'graphql/generated/star';
import { vote } from 'graphql/generated/vote';
import { starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { StarFor, VoteFor } from "@local/shared";
import { ListMenu, ReportDialog } from "..";
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
import { mutationWrapper } from "graphql/utils/wrappers";

/**
 * [label, Icon, iconColor, preview]
 */
const allOptionsMap: { [key in BaseObjectAction]: [string, SvgIconComponent, string, boolean] } = ({
    [BaseObjectAction.Copy]: ['Copy', CopyIcon, 'default', true],
    [BaseObjectAction.Delete]: ['Delete', DeleteIcon, "default", false],
    [BaseObjectAction.Donate]: ['Donate', DonateIcon, "default", true],
    [BaseObjectAction.Downvote]: ['Downvote', DownvoteIcon, "default", false],
    [BaseObjectAction.Edit]: ['Edit', EditIcon, "default", false],
    [BaseObjectAction.Fork]: ['Fork', ForkIcon, "default", true],
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
    handleActionComplete,
    handleDelete,
    handleEdit,
    objectId,
    objectType,
    title,
    anchorEl,
    availableOptions,
    onClose,
    session,
}: BaseObjectActionDialogProps) => {
    // States
    const [donateOpen, setDonateOpen] = useState(false);
    const [shareOpen, setShareOpen] = useState(false);
    const [reportOpen, setReportOpen] = useState(false);

    const openDonate = useCallback(() => setDonateOpen(true), [setDonateOpen]);
    const closeDonate = useCallback(() => setDonateOpen(false), [setDonateOpen]);

    const openShare = useCallback(() => setShareOpen(true), [setShareOpen]);
    const closeShare = useCallback(() => setShareOpen(false), [setShareOpen]);

    const openReport = useCallback(() => setReportOpen(true), [setReportOpen]);
    const closeReport = useCallback(() => setReportOpen(false), [setReportOpen]);

    // Mutations
    //const [copy] = useMutation<copy>(copyMutation);
    //const [delete] = useMutation<delete>(deleteMutation);
    //const [fork] = useMutation<fork>(forkMutation);
    const [star] = useMutation<star>(starMutation);
    const [vote] = useMutation<vote>(voteMutation);

    const handleStar = useCallback((isStar: boolean, starFor: StarFor) => {
        mutationWrapper({
            mutation: star,
            input: { isStar, starFor, forId: objectId },
        })
    }, [objectId]);

    const handleVote = useCallback((isUpvote: boolean | null, voteFor: VoteFor) => {
        mutationWrapper({
            mutation: vote,
            input: { isUpvote, voteFor, forId: objectId },
        })
    }, [objectId]);

    const onSelect = useCallback((action: BaseObjectAction) => {
        switch (action) {
            case BaseObjectAction.Copy:
                //copy({ variables: { input: { id: objectId } } });
                break;
            case BaseObjectAction.Delete:
                //delete({ variables: { input: { id: objectId } } });
                break;
            case BaseObjectAction.Donate:
                openDonate();
                break;
            case BaseObjectAction.Downvote:
                handleVote(false, objectType as VoteFor);
                break;
            case BaseObjectAction.Edit:
                //edit({ variables: { input: { id: objectId } } });
                break;
            case BaseObjectAction.Fork:
                //fork({ variables: { input: { id: objectId } } });
                break;
            case BaseObjectAction.Report:
                openReport();
                break;
            case BaseObjectAction.Share:
                openShare();
                break;
            case BaseObjectAction.Star:
                handleStar(true, objectType as StarFor);
                break;
            case BaseObjectAction.Unstar:
                handleStar(false, objectType as StarFor);
                break;
            case BaseObjectAction.Upvote:
                handleVote(true, objectType as VoteFor);
                break;
        }
        onClose();
    }, [onClose]);

    const data: ListMenuItemData<BaseObjectAction>[] = useMemo(() => {
        // Convert options to ListMenuItemData
        return availableOptions
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
    }, []);

    return (
        <>
            {/* Report dialog */}
            <ReportDialog
                reportFor={objectType}
                forId={objectId}
                open={reportOpen}
                onClose={closeReport}
                session={session}
            />
            {/* Actual action dialog */}
            <ListMenu
                id={`${objectType}-options-menu-${objectId}`}
                anchorEl={anchorEl}
                title={title}
                data={data}
                onSelect={onSelect}
                onClose={onClose}
            />
        </>
    )
}