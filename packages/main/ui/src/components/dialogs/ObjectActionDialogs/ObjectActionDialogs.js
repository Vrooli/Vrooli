import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useContext } from "react";
import { getDisplay } from "../../../utils/display/listTools";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { SessionContext } from "../../../utils/SessionContext";
import { DeleteDialog } from "../DeleteDialog/DeleteDialog";
import { ReportDialog } from "../ReportDialog/ReportDialog";
import { ShareObjectDialog } from "../ShareObjectDialog/ShareObjectDialog";
import { StatsDialog } from "../StatsDialog/StatsDialog";
export const ObjectActionDialogs = ({ hasBookmarkingSupport, hasCopyingSupport, hasDeletingSupport, hasReportingSupport, hasSharingSupport, hasStatsSupport, hasVotingSupport, isDeleteDialogOpen, isDonateDialogOpen, isShareDialogOpen, isStatsDialogOpen, isReportDialogOpen, onActionStart, onActionComplete, openDeleteDialog, closeDeleteDialog, openDonateDialog, closeDonateDialog, openShareDialog, closeShareDialog, openStatsDialog, closeStatsDialog, openReportDialog, closeReportDialog, object, objectType, zIndex, }) => {
    const session = useContext(SessionContext);
    console.log("isDeleteDialogOpen", isDeleteDialogOpen, object?.id, hasDeletingSupport);
    return (_jsxs(_Fragment, { children: [object?.id && hasDeletingSupport && _jsx(DeleteDialog, { isOpen: isDeleteDialogOpen, objectId: object.id, objectType: objectType, objectName: getDisplay(object, getUserLanguages(session)).title, handleClose: closeDeleteDialog, zIndex: zIndex + 1 }), object?.id && hasReportingSupport && _jsx(ReportDialog, { forId: object.id, onClose: closeReportDialog, open: isReportDialogOpen, reportFor: objectType, zIndex: zIndex + 1 }), hasSharingSupport && _jsx(ShareObjectDialog, { object: object, open: isShareDialogOpen, onClose: closeShareDialog, zIndex: zIndex + 1 }), hasStatsSupport && _jsx(StatsDialog, { handleObjectUpdate: () => { }, isOpen: isStatsDialogOpen, object: object, onClose: closeStatsDialog, zIndex: zIndex + 1 })] }));
};
//# sourceMappingURL=ObjectActionDialogs.js.map