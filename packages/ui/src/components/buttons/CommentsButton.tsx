import { getObjectUrl } from "@local/shared";
import { Box, ListItemText, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { multiLineEllipsis } from "../../styles.js";
import { type CommentsButtonProps } from "./types.js";

export function CommentsButton({
    commentsCount = 0,
    disabled = false,
    object,
}: CommentsButtonProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // When clicked, navigate to object's comment section
    const link = useMemo(() => object ? `${getObjectUrl(object)}#comments` : "", [object]);
    const handleClick = useCallback((event: any) => {
        // Stop propagation to prevent list item from being selected
        event.stopPropagation();
        // Prevent default to stop href from being used
        event.preventDefault();
        if (link.length === 0) return;
        setLocation(link);
    }, [link, setLocation]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                pointerEvents: "none",
            }}
        >
            <Box
                aria-label={t("Comment", { count: commentsCount ?? 0 })}
                component="a"
                href={link}
                onClick={handleClick}
                sx={{
                    display: "contents",
                    cursor: disabled ? "none" : "pointer",
                    pointerEvents: disabled ? "none" : "all",
                }}>
                <IconCommon
                    decorative
                    fill={disabled ? "rgb(189 189 189)" : palette.secondary.main}
                    name="Comment"
                />
            </Box>
            <ListItemText
                primary={commentsCount}
                sx={{ ...multiLineEllipsis(1), pointerEvents: "none" }}
            />
        </Stack>
    );
}
