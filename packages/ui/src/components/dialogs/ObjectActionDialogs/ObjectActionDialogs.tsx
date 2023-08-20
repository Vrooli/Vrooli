import { BookmarkFor, DeleteType, ReportFor } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { useContext } from "react";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { ReportUpsert } from "views/objects/report";
import { StatsObjectView } from "views/StatsObjectView/StatsObjectView";
import { DeleteDialog } from "../DeleteDialog/DeleteDialog";
import { SelectBookmarkListDialog } from "../SelectBookmarkListDialog/SelectBookmarkListDialog";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
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
    isBookmarkDialogOpen,
    isDonateDialogOpen,
    isShareDialogOpen,
    isStatsDialogOpen,
    isReportDialogOpen,
    onActionStart,
    onActionComplete,
    closeBookmarkDialog,
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
}: ObjectActionDialogsProps) => {
    const session = useContext(SessionContext);

    return (
        <>
            {/* openAddCommentDialog?: () => void; //TODO: implement
    openDonateDialog?: () => void; //TODO: implement
    */}
            {object?.id && hasBookmarkingSupport && <SelectBookmarkListDialog
                objectId={object.id}
                objectType={objectType as unknown as BookmarkFor}
                onClose={closeBookmarkDialog}
                isCreate={true}
                isOpen={isBookmarkDialogOpen}
            />}
            {object?.id && hasDeletingSupport && <DeleteDialog
                isOpen={isDeleteDialogOpen}
                objectId={object.id}
                objectType={objectType as unknown as DeleteType}
                objectName={getDisplay(object, getUserLanguages(session)).title}
                handleClose={closeDeleteDialog}
            />}
            {object?.id && hasReportingSupport && <ReportUpsert
                isCreate={true}
                isOpen={isReportDialogOpen}
                onCancel={closeReportDialog}
                onCompleted={closeReportDialog}
                overrideObject={{ createdFor: { __typename: objectType as unknown as ReportFor, id: object.id } }}
            />}
            {hasSharingSupport && <ShareObjectDialog
                object={object}
                open={isShareDialogOpen}
                onClose={closeShareDialog}
            />}
            {hasStatsSupport && <StatsObjectView
                handleObjectUpdate={() => { }} //TODO
                isOpen={isStatsDialogOpen}
                object={object as any}
                onClose={closeStatsDialog}
            />}
        </>
    );
};
