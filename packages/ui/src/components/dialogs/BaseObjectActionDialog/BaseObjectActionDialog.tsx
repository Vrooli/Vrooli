import { useMutation } from "@apollo/client";
import { star } from 'graphql/generated/star';
import { vote } from 'graphql/generated/vote';
import { starMutation, voteMutation } from "graphql/mutation";
import { useCallback, useMemo, useState } from "react";
import { StarFor, VoteFor } from "@local/shared";
import { ListMenu, ReportDialog } from "..";
import { BaseObjectActionDialogProps, BaseObjectAction, ListMenuItemData } from "../types";
import {
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
    SvgIconComponent,
} from "@mui/icons-material";
import { mutationWrapper } from "graphql/utils/wrappers";

/**
 * [label, Icon, iconColor]
 */
const allOptionsMap: { [x: string]: [string, SvgIconComponent, string] } = ({
    [BaseObjectAction.Copy]: ['Copy', CopyIcon, 'default'],
    [BaseObjectAction.Delete]: ['Delete', DeleteIcon, "default"],
    [BaseObjectAction.Donate]: ['Donate', DonateIcon, "default"],
    [BaseObjectAction.Downvote]: ['Downvote', DownvoteIcon, "default"],
    [BaseObjectAction.Edit]: ['Edit', EditIcon, "default"],
    [BaseObjectAction.Fork]: ['Fork', ForkIcon, "default"],
    [BaseObjectAction.Report]: ['Report', ReportIcon, "default"],
    [BaseObjectAction.Share]: ['Share', ShareIcon, "default"],
    [BaseObjectAction.Star]: ['Star', StarIcon, "#cbae30"],
    [BaseObjectAction.Unstar]: ['Unstar', UnstarIcon, "#cbae30"],
    [BaseObjectAction.Upvote]: ['Upvote', UpvoteIcon, "default"],
})

export const BaseObjectActionDialog = ({
    objectId,
    objectType,
    title,
    anchorEl,
    availableOptions,
    onClose,
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
                const [label, Icon, iconColor] = allOptionsMap[option];
                return {
                    label,
                    value: option,
                    Icon,
                    iconColor,
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