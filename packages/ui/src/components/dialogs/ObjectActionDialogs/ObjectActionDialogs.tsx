import { DeleteType, ReportFor } from "@shared/consts";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { DeleteDialog } from "../DeleteDialog/DeleteDialog";
import { ReportDialog } from "../ReportDialog/ReportDialog";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
import { StatsDialog } from "../StatsDialog/StatsDialog";
import { ObjectActionDialogsProps } from "../types";

export const ObjectActionDialogs = ({
    hasBookmarkingSupport,
    hasCopyingSupport,
    hasDeletingSupport,
    hasReportingSupport,
    hasSharingSupport,
    hasStatsSupport,
    hasVotingSupport,
    isDeleteDialogOpen,
    isDonateDialogOpen,
    isShareDialogOpen,
    isStatsDialogOpen,
    isReportDialogOpen,
    onActionStart,
    onActionComplete,
    openDeleteDialog,
    closeDeleteDialog,
    openDonateDialog,
    closeDonateDialog,
    openShareDialog,
    closeShareDialog,
    openStatsDialog,
    closeStatsDialog,
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
            {hasStatsSupport && <StatsDialog
                handleObjectUpdate={() => { }} //TODO
                isOpen={isStatsDialogOpen}
                object={object as any}
                onClose={closeStatsDialog}
                session={session}
                zIndex={zIndex + 1}
            />}
        </>
    )
}