import { BookmarkFor, ReportFor } from "@local/shared";
import { ObjectAction } from "utils/actions/objectActions";
import { ReportUpsert } from "views/objects/report";
import { StatsObjectView } from "views/StatsObjectView/StatsObjectView";
import { SelectBookmarkListDialog } from "../SelectBookmarkListDialog/SelectBookmarkListDialog";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
import { ObjectActionDialogsProps } from "../types";

export const ObjectActionDialogs = ({
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
}: ObjectActionDialogsProps) => {
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
                isCreate={true}
                isOpen={isReportDialogOpen}
                onCancel={closeReportDialog}
                onCompleted={closeReportDialog}
                overrideObject={{ createdFor: { __typename: objectType as unknown as ReportFor, id: object.id } }}
            />}
            {availableActions.includes(ObjectAction.Share) && <ShareObjectDialog
                object={object}
                open={isShareDialogOpen}
                onClose={closeShareDialog}
            />}
            {availableActions.includes(ObjectAction.Stats) && <StatsObjectView
                handleObjectUpdate={() => { }} //TODO
                isOpen={isStatsDialogOpen}
                object={object as any}
                onClose={closeStatsDialog}
            />}
        </>
    );
};
