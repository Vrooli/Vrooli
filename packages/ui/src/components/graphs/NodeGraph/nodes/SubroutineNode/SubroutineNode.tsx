import { CloseIcon, name as nameValidation, reqErr, Routine } from "@local/shared";
import { Box, Checkbox, Container, FormControlLabel, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { EditableLabel } from "components/inputs/EditableLabel/EditableLabel";
import { CSSProperties, useCallback, useMemo, useState } from "react";
import { multiLineEllipsis, noSelect, textShadow } from "styles";
import { BuildAction } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { updateTranslationFields } from "utils/display/translationTools";
import usePress from "utils/hooks/usePress";
import { calculateNodeSize } from "..";
import { NodeContextMenu } from "../../NodeContextMenu/NodeContextMenu";
import { routineNodeCheckboxLabel, routineNodeCheckboxOption } from "../styles";
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
    const fontSize = useMemo(() => `min(${calculateNodeSize(220, scale) / 5}px, 2em)`, [scale]);
    // Determines if the subroutine is one you can edit
    const canUpdate = useMemo<boolean>(() => ((data?.routineVersion?.root as Routine)?.isInternal ?? (data?.routineVersion?.root as Routine)?.you?.canUpdate === true), [data.routineVersion]);

    const { title } = useMemo(() => getDisplay(data, navigator.languages), [data]);

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

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate(data.id, {
            ...data,
            translations: updateTranslationFields(data, language, { name: newLabel }),
        });
    }, [handleUpdate, data, language]);

    const onOptionalChange = useCallback((checked: boolean) => {
        handleUpdate(data.id, {
            ...data,
            isOptional: checked,
        });
    }, [handleUpdate, data]);

    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canUpdate={isEditing}
                handleUpdate={handleLabelUpdate}
                renderLabel={(t) => (
                    <Typography
                        id={`subroutine-name-${data.id}`}
                        variant="h6"
                        sx={{
                            ...noSelect,
                            ...textShadow,
                            ...multiLineEllipsis(1),
                            textAlign: "center",
                            width: "100%",
                            lineBreak: "anywhere" as any,
                            whiteSpace: "pre" as any,
                        } as CSSProperties}
                    >{firstString(t, "Untitled")}</Typography>
                )}
                sxs={{
                    stack: {
                        marginLeft: "auto",
                        marginRight: "auto",
                    },
                }}
                text={title}
                validationSchema={nameValidation.required(reqErr)}
                zIndex={zIndex}
            />
        );
    }, [labelVisible, isEditing, handleLabelUpdate, title, zIndex, data.id]);

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
                    boxShadow: 12,
                    minWidth: nodeSize,
                    fontSize,
                    position: "relative",
                    display: "block",
                    marginBottom: "8px",
                    borderRadius: "12px",
                    overflow: "hidden",
                    backgroundColor: palette.background.paper,
                    color: palette.background.textPrimary,
                }}
            >
                <Container
                    id={`subroutine-name-bar-${data.id}`}
                    {...pressEvents}
                    aria-owns={contextOpen ? contextId : undefined}
                    sx={{
                        display: "flex",
                        alignItems: "center",
                        backgroundColor: canUpdate ?
                            (palette.mode === "light" ? palette.primary.dark : palette.secondary.dark) :
                            "#667899",
                        color: palette.mode === "light" ? palette.primary.contrastText : palette.secondary.contrastText,
                        padding: "0.1em",
                        textAlign: "center",
                        cursor: "pointer",
                        "&:hover": {
                            filter: "brightness(120%)",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    {labelObject}
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
                <Stack direction="row" justifyContent="space-between" borderRadius={0} sx={{ ...noSelect }}>
                    <Tooltip placement={"top"} title='Routine can be skipped'>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Optional'
                            control={
                                <Checkbox
                                    id={`${title}-optional-option`}
                                    size="small"
                                    name='isOptionalCheckbox'
                                    color='secondary'
                                    checked={data?.isOptional}
                                    onChange={(_e, checked) => { onOptionalChange(checked); }}
                                    onTouchStart={() => { onOptionalChange(!data?.isOptional); }}
                                    sx={{ ...routineNodeCheckboxOption }}
                                />
                            }
                            sx={{ ...routineNodeCheckboxLabel }}
                        />
                    </Tooltip>
                    {/* <Typography variant="body2">Steps: 4</Typography> */}
                </Stack>
            </Box>
        </>
    );
};
