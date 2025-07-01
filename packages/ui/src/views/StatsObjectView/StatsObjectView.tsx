import Box from "@mui/material/Box";
import { type ListObject } from "@vrooli/shared";
import { useTranslation } from "react-i18next";
import Dialog from "@mui/material/Dialog";
import { UpTransition } from "../../components/transitions/UpTransition/UpTransition.js";
import { useIsMobile } from "../../hooks/useIsMobile.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { StatsCompact } from "../../components/text/StatsCompact.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { type StatsObjectViewProps } from "../types.js";

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
    const isMobile = useIsMobile();

    return display === "Dialog" ? (
        <Dialog
            id="object-stats-dialog"
            onClose={onClose}
            open={isOpen}
            aria-labelledby={titleId}
            TransitionComponent={isMobile ? UpTransition : undefined}
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
        </Dialog>
    ) : (
        <>
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
        </>
    );
}
