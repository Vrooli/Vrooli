import Avatar from "@mui/material/Avatar";
import Chip from "@mui/material/Chip";
import { styled, useTheme } from "@mui/material/styles";
import type { Theme } from "@mui/material/styles";
import { type CSSProperties } from "@mui/styles";
import React, { useMemo } from "react";
import { Icon, IconCommon, type IconInfo } from "../../../../icons/Icons.js";
import { IconButton as CustomIconButton } from "../../../buttons/IconButton.js";
import type { ContextItem } from "../utils.js";
import {
    CODE_FILE_REGEX,
    ENV_FILE_REGEX,
    EXECUTABLE_FILE_REGEX,
    IMAGE_FILE_REGEX,
    MAX_LABEL_LENGTH,
    TEXT_FILE_REGEX,
    VIDEO_FILE_REGEX,
    truncateLabel,
} from "./contextUtils.js";

const PreviewContainer = styled("div")(({ theme }) => ({
    position: "relative",
    width: theme.spacing(7),
    height: theme.spacing(7),
    borderRadius: theme.spacing(1),
    overflow: "visible",
    marginRight: theme.spacing(1),
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));

export function previewImageStyle(theme: Theme) {
    return {
        width: "100%",
        height: "100%",
        objectFit: "cover",
        borderRadius: theme.spacing(1),
        border: `1px solid ${theme.palette.divider}`,
    } as const;
}

const PreviewImageAvatar = styled(Avatar)(({ theme }) => ({
    width: "100%",
    height: "100%",
    borderRadius: theme.spacing(1),
    border: `1px solid ${theme.palette.divider}`,
}));

// RemoveIconButton is now a wrapper component instead of styled component
const RemoveIconButton = ({ onClick, children }: { onClick: (event: React.MouseEvent) => void, children: React.ReactNode }) => {
    const theme = useTheme();
    return (
        <div className="tw-absolute -tw-top-1 -tw-right-1">
            <CustomIconButton
                size={16}
                onClick={onClick}
                variant="solid"
                className="tw-p-0.5"
            >
                {children}
            </CustomIconButton>
        </div>
    );
};

const ContextItemChip = styled(Chip)(({ theme }) => ({
    marginRight: theme.spacing(1),
    marginBottom: theme.spacing(1),
}));

type ContextItemDisplayProps = {
    imgStyle: CSSProperties;
    item: ContextItem;
    onRemove: (id: string) => unknown;
};

export function ContextItemDisplay({
    imgStyle,
    item,
    onRemove,
}: ContextItemDisplayProps) {
    function handleRemove(event: React.MouseEvent) {
        // Stop propagation to prevent Outer click handler
        event.stopPropagation();
        onRemove(item.id);
    }

    const fallbackIconInfo = useMemo<IconInfo>(function fallbackInfoBasedOnTypeMemo() {
        if (item.type === "image") return { name: "Image", type: "Common" } as const;
        if (item.type === "text") return { name: "Article", type: "Common" } as const;
        // For files, check if it's a text-based file
        if (item.type === "file" && item.file?.type) {
            // Image files
            if (item.file.type.startsWith("image/") || IMAGE_FILE_REGEX.test(item.file.name)) {
                return { name: "Image", type: "Common" } as const;
            }
            // Text/document files
            if (item.file.type.startsWith("text/") || TEXT_FILE_REGEX.test(item.file.name)) {
                return { name: "Article", type: "Common" } as const;
            }
            // Code files
            if (item.file.type.includes("javascript") ||
                item.file.type.includes("json") ||
                CODE_FILE_REGEX.test(item.file.name)) {
                return { name: "Object", type: "Common" } as const;
            }
            // Video files
            if (item.file.type.startsWith("video/") || VIDEO_FILE_REGEX.test(item.file.name)) {
                return { name: "SocialVideo", type: "Common" } as const;
            }
            // Environment files
            if (ENV_FILE_REGEX.test(item.file.name)) {
                return { name: "Invisible", type: "Common" } as const;
            }
            // Executable files
            if (EXECUTABLE_FILE_REGEX.test(item.file.name)) {
                return { name: "Terminal", type: "Common" } as const;
            }
        }
        return { name: "File", type: "Common" } as const;
    }, [item.type, item.file?.type, item.file?.name]);

    // Check if this is an image that should be displayed as a preview
    const shouldShowPreview = useMemo(() => {
        if (item.type === "image") return true;
        if (item.type === "file" && item.file?.type?.startsWith("image/")) return true;
        if (item.type === "file" && IMAGE_FILE_REGEX.test(item.file?.name ?? "")) return true;
        return false;
    }, [item.type, item.file?.type, item.file?.name]);

    const truncatedLabel = useMemo(() => truncateLabel(item.label, MAX_LABEL_LENGTH), [item.label]);

    if (shouldShowPreview) {
        return (
            <PreviewContainer data-type="context-item"> {/* Add data-type */}
                {item.src ? (
                    <img src={item.src} alt={item.label} style={imgStyle} />
                ) : (
                    <PreviewImageAvatar variant="square">
                        <Icon
                            decorative
                            info={fallbackIconInfo}
                        />
                    </PreviewImageAvatar>
                )}
                {/* Ensure the remove button stops propagation */}
                <RemoveIconButton onClick={handleRemove}>
                    <IconCommon
                        decorative
                        fill="background.default"
                        name="Close"
                    />
                </RemoveIconButton>
            </PreviewContainer>
        );
    }

    return (
        <ContextItemChip
            data-type="context-item" // Add data-type
            icon={<Icon decorative info={fallbackIconInfo} />}
            label={truncatedLabel}
            onDelete={handleRemove} // MuiChip handles stopPropagation for deleteIcon
            title={item.label} // Show full name on hover
        // No onClick needed here, delete is handled
        />
    );
}
