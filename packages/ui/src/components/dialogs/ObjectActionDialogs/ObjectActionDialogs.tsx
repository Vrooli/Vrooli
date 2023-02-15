import { DeleteType, ReportFor } from "@shared/consts";
import { DeleteDialog, ReportDialog } from "..";
import { ObjectActionDialogsProps } from "../types";
import { getDisplay, getUserLanguages } from "utils";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";

export const ObjectActionDialogs = ({
    hasBookmarkingSupport,
    hasCopyingSupport,
    hasDeletingSupport,
    hasReportingSupport,
    hasSharingSupport,
    hasVotingSupport,
    isDeleteDialogOpen,
    isDonateDialogOpen,
    isShareDialogOpen,
    isReportDialogOpen,
    onActionStart,
    onActionComplete,
    openDeleteDialog,
    closeDeleteDialog,
    openDonateDialog,
    closeDonateDialog,
    openShareDialog,
    closeShareDialog,
    openReportDialog,
    closeReportDialog,
    object,
    objectType,
    session,
    zIndex,
}: ObjectActionDialogsProps) => {
    return (
        <>
        {/* openAddCommentDialog?: () => void; //TODO: implement
    openDonateDialog?: () => void; //TODO: implement
    */}
            {object?.id && hasDeletingSupport && <DeleteDialog
                isOpen={isDeleteDialogOpen}
                objectId={object.id}
                objectType={objectType as unknown as DeleteType}
                objectName={getDisplay(object, getUserLanguages(session)).title}
                handleClose={closeDeleteDialog}
                zIndex={zIndex + 1}
            />}
            {object?.id && hasReportingSupport && <ReportDialog
                forId={object.id}
                onClose={closeReportDialog}
                open={isReportDialogOpen}
                reportFor={objectType as unknown as ReportFor}
                session={session}
                zIndex={zIndex + 1}
            />}
            {hasSharingSupport && <ShareObjectDialog
                object={object}
                open={isShareDialogOpen}
                onClose={closeShareDialog}
                zIndex={zIndex + 1}
            />}
        </>
    )
}