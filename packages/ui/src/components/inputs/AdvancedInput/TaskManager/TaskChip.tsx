import Chip from "@mui/material/Chip";
import { useTheme } from "@mui/material/styles";
import React, { useCallback, useMemo, useState } from "react";
import { Icon, IconCommon } from "../../../../icons/Icons.js";
import { AITaskDisplayState, type AITaskDisplay } from "../../../../types.js";
import { IconButton as CustomIconButton } from "../../../buttons/IconButton.js";

const toolChipIconButtonClassName = "tw-p-0 tw-pr-0.5" as const;

export type TaskChipProps = AITaskDisplay & {
    index: number;
    onRemoveTool: () => unknown;
    onToggleTool: () => unknown;
    onToggleToolExclusive: () => unknown;
}

export function TaskChip({
    displayName,
    iconInfo,
    index,
    name,
    onRemoveTool,
    onToggleTool,
    onToggleToolExclusive,
    state,
}: TaskChipProps) {
    const { palette } = useTheme();

    const [isHovered, setIsHovered] = useState(false);
    function handleMouseEnter() {
        setIsHovered(true);
    }
    function handleMouseLeave() {
        setIsHovered(false);
    }

    const handlePlayClick = useCallback(function handlePlayClickCallback(event: React.MouseEvent) {
        // Prevent triggering chip's onClick
        event.stopPropagation();
        onToggleToolExclusive();
    }, [onToggleToolExclusive]);

    const handleToolToggle = useCallback(function handleToolToggleCallback(event: React.MouseEvent) {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        onToggleTool();
    }, [onToggleTool]);

    const chipVariant = state === AITaskDisplayState.Disabled ? "outlined" : "filled";
    const chipStyle = useMemo(() => {
        let backgroundColor: string;
        let color: string;
        let border: string;

        switch (state) {
            case AITaskDisplayState.Disabled: {
                backgroundColor = "transparent";
                color = palette.background.textSecondary;
                border = `1px solid ${palette.divider}`;
                break;
            }
            case AITaskDisplayState.Enabled: {
                backgroundColor = palette.secondary.light,
                    color = palette.secondary.contrastText,
                    border = "none";
                break;
            }
            case AITaskDisplayState.Exclusive: {
                backgroundColor = palette.secondary.main;
                color = palette.secondary.contrastText;
                border = "none";
                break;
            }
        }
        return {
            backgroundColor,
            color,
            border,
            cursor: "pointer",
            "&:hover": {
                backgroundColor,
                color,
                filter: "brightness(1.05)",
            },
        } as const;
    }, [state, palette]);

    const chipIcon = useMemo(() => {
        if (isHovered) {
            if (state !== AITaskDisplayState.Exclusive) {
                return (
                    <CustomIconButton
                        size={20}
                        onClick={handlePlayClick}
                        className={toolChipIconButtonClassName}
                    >
                        <IconCommon
                            decorative
                            name="Play"
                        />
                    </CustomIconButton>
                );
            }
            return (
                <CustomIconButton
                    size={20}
                    onClick={handlePlayClick}
                    className={toolChipIconButtonClassName}
                >
                    <IconCommon
                        decorative
                        name="Pause"
                    />
                </CustomIconButton>
            );
        }
        return <Icon decorative info={iconInfo} />;
    }, [isHovered, iconInfo, state, handlePlayClick]);

    return (
        <Chip
            data-type="task" // Keep this for the selector
            key={`${name}-${index}`}
            icon={chipIcon}
            label={displayName}
            onDelete={onRemoveTool} // MuiChip handles stopPropagation internally for deleteIcon
            onClick={handleToolToggle}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            sx={chipStyle}
            variant={chipVariant}
        />
    );
}
