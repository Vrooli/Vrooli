import { Box } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { StatsCompactPropsObject } from "components/text/types";
import { useTranslation } from "react-i18next";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { StatsDialogProps } from "../types";

const titleId = "stats-object-dialog-title";

/**
 * Displays basic stats about an object, in a short format.
 * Displays votes, views, date created, and reports
 */
export const StatsDialog = <T extends StatsCompactPropsObject>({
    handleObjectUpdate,
    isOpen,
    object,
    onClose,
    zIndex,
}: StatsDialogProps<T>) => {
    const { t } = useTranslation();

    return (
        <LargeDialog
            id="object-stats-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("Share")}
                titleId={titleId}
            />
            <Box sx={{ padding: 2 }}>
                {/* Bookmarks, votes, and other info */}
                <StatsCompact
                    handleObjectUpdate={handleObjectUpdate}
                    loading={false}
                    object={object}
                />
                {/* Historical stats */}
                {/* TODO */}
            </Box>
        </LargeDialog>
    );
};
