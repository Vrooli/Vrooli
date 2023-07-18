import { Routine } from "@local/shared";
import { Box, Container, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { ActionIcon, CloseIcon, NoActionIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { multiLineEllipsis, noSelect, textShadow } from "styles";
import { BuildAction } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import usePress from "utils/hooks/usePress";
import { calculateNodeSize } from "..";
import { NodeWidth } from "../..";
import { NodeContextMenu } from "../../NodeContextMenu/NodeContextMenu";
import { routineNodeActionStyle, routineNodeCheckboxOption } from "../styles";
import { SubroutineNodeProps } from "../types";

/**
 * Decides if a clicked element should trigger opening the subroutine dialog 
 * @param id ID of the clicked element
 */
const shouldOpen = (id: string | null | undefined): boolean => {
    // Only collapse if clicked on name bar or name
    return Boolean(id && (id.startsWith("subroutine-name-")));
};

export const SubroutineNode = ({
    data,
    scale = 1,
    labelVisible = true,
    isOpen,
    isEditing = true,
    handleAction,
    handleUpdate,
    language,
    zIndex,
}: SubroutineNodeProps) => {
    const { palette } = useTheme();

    const nodeSize = useMemo(() => `${calculateNodeSize(220, scale)}px`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 8}px, 1.5em)`, [scale]);
    // Determines if the subroutine is one you can edit
    const canUpdate = useMemo<boolean>(() => ((data?.routineVersion?.root as Routine)?.isInternal ?? (data?.routineVersion?.root as Routine)?.you?.canUpdate === true), [data.routineVersion]);

    const { title } = useMemo(() => getDisplay({ ...data, __typename: "NodeRoutineListItem" }, navigator.languages), [data]);

    const onAction = useCallback((event: any | null, action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine) => {
        if (event && [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine].includes(action)) {
            event.stopPropagation();
        }
        handleAction(action, data.id);
    }, [data.id, handleAction]);
    const openSubroutine = useCallback((target: EventTarget) => {
        if (!shouldOpen(target.id)) return;
        onAction(null, BuildAction.OpenSubroutine);
    }, [onAction]);
    const deleteSubroutine = useCallback((event: any) => { onAction(event, BuildAction.DeleteSubroutine); }, [onAction]);

    const onOptionalChange = useCallback((checked: boolean) => {
        if (!isEditing) return;
        handleUpdate(data.id, {
            ...data,
            isOptional: checked,
        });
    }, [isEditing, handleUpdate, data]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `subroutine-context-menu-${data.id}`, [data?.id]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not editing
        if (!isEditing) return;
        setContextAnchor(target);
    }, [isEditing]);
    const closeContext = useCallback(() => { setContextAnchor(null); }, []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openSubroutine,
        onRightClick: openContext,
    });

    return (
        <>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={
                    isEditing ?
                        [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine] :
                        [BuildAction.OpenSubroutine, BuildAction.DeleteSubroutine]
                }
                handleClose={closeContext}
                handleSelect={(action) => { onAction(null, action as BuildAction.EditSubroutine | BuildAction.DeleteSubroutine | BuildAction.OpenSubroutine); }}
                zIndex={zIndex + 1}
            />
            <Box
                sx={{
                    boxShadow: 6,
                    minWidth: nodeSize,
                    position: "relative",
                    display: "block",
                    marginBottom: "8px",
                    borderRadius: "12px",
                    overflow: "hidden",
                    background: palette.mode === "light" ? "#b0bbe7" : "#384164",
                    color: palette.background.textPrimary,
                    "@media print": {
                        border: `1px solid ${palette.mode === "light" ? "#b0bbe7" : "#384164"}`,
                        boxShadow: "none",
                    },
                }}
            >
                <Container
                    id={`subroutine-name-bar-${data.id}`}
                    {...pressEvents}
                    aria-owns={contextOpen ? contextId : undefined}
                    sx={{
                        display: "flex",
                        alignItems: "center",
                        background: canUpdate ?
                            (palette.mode === "light" ? palette.primary.dark : palette.secondary.dark) :
                            "#667899",
                        color: palette.mode === "light" ? palette.primary.contrastText : palette.secondary.contrastText,
                        padding: "0.1em",
                        paddingLeft: "8px!important",
                        paddingRight: "8px!important",
                        textAlign: "center",
                        cursor: "pointer",
                        "&:hover": {
                            filter: "brightness(120%)",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    <Tooltip placement={"top"} title='Routine can be skipped'>
                        <Box
                            onClick={() => { onOptionalChange(!data?.isOptional); }}
                            onTouchStart={() => { onOptionalChange(!data?.isOptional); }}
                            sx={routineNodeActionStyle(isEditing)}
                        >
                            <IconButton
                                size="small"
                                sx={{ ...routineNodeCheckboxOption }}
                                disabled={!isEditing}
                            >
                                {data?.isOptional ? <NoActionIcon /> : <ActionIcon />}
                            </IconButton>
                        </Box>
                    </Tooltip>
                    {labelVisible && <Typography
                        id={`subroutine-name-${data.id}`}
                        variant="h6"
                        sx={{
                            ...noSelect,
                            ...textShadow,
                            ...multiLineEllipsis(2),
                            fontSize,
                            textAlign: "center",
                            width: "100%",
                            lineBreak: scale < -2 ? "anywhere" : "normal",
                            // whiteSpace: "pre" as any,
                            "@media print": {
                                textShadow: "none",
                                color: "black",
                            },
                        }}
                    >{firstString(title, "Untitled")}</Typography>}
                    {isEditing && (
                        <IconButton
                            id={`subroutine-delete-icon-button-${data.id}`}
                            onClick={deleteSubroutine}
                            onTouchStart={deleteSubroutine}
                            color="inherit"
                        >
                            <CloseIcon id={`subroutine-delete-icon-${data.id}`} />
                        </IconButton>
                    )}
                </Container>
            </Box>
        </>
    );
};
