import { Box } from "@mui/material";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { useTranslation } from "react-i18next";
import { getDisplay, ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { StatsObjectViewProps } from "../types";

const titleId = "stats-object-dialog-title";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsObjectView = <T extends ListObject>({
    handleObjectUpdate,
    isOpen,
    object,
    onClose,
}: StatsObjectViewProps<T>) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    return (
        <MaybeLargeDialog
            display={display}
            id="object-stats-dialog"
            onClose={onClose}
            isOpen={isOpen ?? false}
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
};
