import { ListObject } from "@local/shared";
import { Box } from "@mui/material";
import { useTranslation } from "react-i18next";
import { MaybeLargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { StatsCompact } from "../../components/text/StatsCompact.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { StatsObjectViewProps } from "../types.js";

const titleId = "stats-object-dialog-title";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export function StatsObjectView<T extends ListObject>({
    display,
    handleObjectUpdate,
    isOpen,
    object,
    onClose,
}: StatsObjectViewProps<T>) {
    const { t } = useTranslation();

    return (
        <MaybeLargeDialog
            display={display}
            id="object-stats-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={titleId}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t("ObjectStats", { objectName: getDisplay(object).title })}
                titleId={titleId}
            />
            <Box sx={{ padding: 2 }}>
                {/* Bookmarks, votes, and other info */}
                <StatsCompact
                    handleObjectUpdate={handleObjectUpdate}
                    object={object}
                />
                {/* Historical stats */}
                {/* TODO */}
            </Box>
        </MaybeLargeDialog>
    );
}
