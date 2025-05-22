import { type BookmarkFor, type ReportFor } from "@local/shared";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { StatsObjectView } from "../../../views/StatsObjectView/StatsObjectView.js";
import { ReportUpsert } from "../../../views/objects/report/ReportUpsert.js";
import { SelectBookmarkListDialog } from "../SelectBookmarkListDialog/SelectBookmarkListDialog.js";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog.js";
import { type ObjectActionDialogsProps } from "../types.js";

export function ObjectActionDialogs({
    availableActions,
    isDeleteDialogOpen,
    isBookmarkDialogOpen,
    isDonateDialogOpen,
    isShareDialogOpen,
    isStatsDialogOpen,
    isReportDialogOpen,
    onActionStart,
    onActionComplete,
    closeBookmarkDialog,
    closeDeleteDialog,
    closeDonateDialog,
    closeShareDialog,
    closeStatsDialog,
    closeReportDialog,
    DeleteDialogComponent,
    object,
    objectType,
}: ObjectActionDialogsProps) {
    return (
        <>
            {/* openAddCommentDialog?: () => unknown; //TODO: implement
    openDonateDialog?: () => unknown; //TODO: implement
    */}
            {object?.id && (availableActions.includes(ObjectAction.Bookmark) || availableActions.includes(ObjectAction.BookmarkUndo)) && <SelectBookmarkListDialog
                objectId={object.id}
                objectType={objectType as unknown as BookmarkFor}
                onClose={closeBookmarkDialog}
                isCreate={true}
                isOpen={isBookmarkDialogOpen}
            />}
            {object?.id && DeleteDialogComponent}
            {object?.id && availableActions.includes(ObjectAction.Delete) && <ReportUpsert
                createdFor={{ __typename: objectType as unknown as ReportFor, id: object.id }}
                display="Dialog"
                isCreate={true}
                isOpen={isReportDialogOpen}
                onCancel={closeReportDialog}
                onClose={closeReportDialog}
                onCompleted={closeReportDialog}
                onDeleted={closeReportDialog}
            />}
            {availableActions.includes(ObjectAction.Share) && <ShareObjectDialog
                object={object}
                open={isShareDialogOpen}
                onClose={closeShareDialog}
            />}
            {availableActions.includes(ObjectAction.Stats) && <StatsObjectView
                display="Dialog"
                handleObjectUpdate={() => { }} //TODO
                isOpen={isStatsDialogOpen}
                object={object as any}
                onClose={closeStatsDialog}
            />}
        </>
    );
}
