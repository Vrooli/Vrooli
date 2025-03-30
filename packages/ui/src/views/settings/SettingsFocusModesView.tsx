import { DeleteOneInput, DeleteType, endpointsActions, FocusMode, FocusModeFilterType, FocusModeStopCondition, LINKS, MaxObjects, noop, SessionUser, Success } from "@local/shared";
import { Box, Button, Chip, Divider, IconButton, ListItem, ListItemProps, ListItemText, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { memo, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { FocusModeInfo, useFocusModesStore } from "../../components/inputs/FocusModeSelector/FocusModeSelector.js";
import { TagSelectorBase } from "../../components/inputs/TagSelector/TagSelector.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { SessionContext } from "../../contexts/session.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { FormSection, multiLineEllipsis, ScrollBox } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { FocusModeUpsert } from "../objects/focusMode/FocusModeUpsert.js";
import { SettingsFocusModesViewProps } from "./types.js";

const TitleStack = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    paddingTop: theme.spacing(2),
}));
const AddIconButton = styled(IconButton)(({ theme }) => ({
    padding: theme.spacing(1),
}));
interface StyledFocusModeListItemProps extends ListItemProps {
    isActive: boolean;
}
const StyledFocusModeListItem = styled(ListItem, {
    shouldForwardProp: (prop) => prop !== "isActive",
})<StyledFocusModeListItemProps>(({ isActive, theme }) => ({
    backgroundColor: isActive ? theme.palette.action.selected : "inherit",
    cursor: "pointer",
}));
const FocusModeListItemInner = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    gap: theme.spacing(0.5),
    width: "100%",
    pointerEvents: "none",
}));
const FocusModeNameContainer = styled(Box)(() => ({
    display: "flex",
    alignItems: "center",
}));
const FocusModeName = styled(Typography)(({ theme }) => ({
    ...multiLineEllipsis(1),
    lineBreak: "anywhere",
    pointerEvents: "none",
    paddingRight: theme.spacing(1),
}));
const FocusModeDescription = styled(ListItemText)(({ theme }) => ({
    ...multiLineEllipsis(2),
    color: theme.palette.text.secondary,
    pointerEvents: "none",
}));
const DeleteFocusModeBox = styled(Box)(({ theme }) => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    cursor: "pointer",
    pointerEvents: "all",
    paddingLeft: theme.spacing(1),
}));
const IsActiveChip = styled(Chip)(() => ({
    pointerEvents: "auto",
    width: "fit-content",
}));
const StyledDivider = styled(Divider)(({ theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    margin: theme.spacing(4, 2),
}));
const WarningBox = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    color: theme.palette.warning.main,
    marginBottom: theme.spacing(2),
}));

const focusModeIconStyle = { marginRight: 8 } as const;
const fallbackOverrideObject = { __typename: "FocusMode" } as const;
const listContainerStyle = { maxWidth: "500px" } as const;
const warningIconStyle = { fontSize: 20, marginRight: "8px" } as const;
const updateActiveFocusModeButtonStyle = { marginTop: "16px" } as const;

export function SettingsFocusModesView({
    display,
    onClose,
}: SettingsFocusModesViewProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const getFocusModeInfo = useFocusModesStore(state => state.getFocusModeInfo);
    const setFocusModes = useFocusModesStore(state => state.setFocusModes);
    const putActiveFocusMode = useFocusModesStore(state => state.putActiveFocusMode);
    const [focusModeInfo, setFocusModeInfo] = useState<FocusModeInfo>({ active: null, all: [] });
    useEffect(function fetchFocusModeInfoEffect() {
        const abortController = new AbortController();

        async function fetchFocusModeInfo() {
            const info = await getFocusModeInfo(session, abortController.signal);
            setFocusModeInfo(info);
        }

        fetchFocusModeInfo();

        return () => {
            abortController.abort();
        };
    }, [getFocusModeInfo, session]);

    const { canAdd, hasPremium } = useMemo(() => {
        const { hasPremium } = getCurrentUser(session);
        const max = hasPremium ? MaxObjects.FocusMode.User.premium : MaxObjects.FocusMode.User.noPremium;
        return { canAdd: focusModeInfo.all.length < max, hasPremium };
    }, [focusModeInfo.all.length, session]);

    const [deleteMutation] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback((focusMode: FocusMode) => {
        // Don't delete if there is only one focus mode left
        if (focusModeInfo.all.length === 1) {
            PubSub.get().publish("snack", { messageKey: "MustHaveFocusMode", severity: "Error" });
            return;
        }
        // Confirmation dialog
        PubSub.get().publish("alertDialog", {
            messageKey: "DeleteConfirm",
            buttons: [
                {
                    labelKey: "Yes", onClick: () => {
                        fetchLazyWrapper<DeleteOneInput, Success>({
                            fetch: deleteMutation,
                            inputs: { id: focusMode.id, objectType: DeleteType.FocusMode },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ messageKey: "FocusModeDeleted" }),
                            onSuccess: () => {
                                setFocusModes((prevFocusModes) => prevFocusModes.filter((prevFocusMode) => prevFocusMode.id !== focusMode.id));
                            },
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [deleteMutation, focusModeInfo.all.length, setFocusModes]);

    const handleActivate = useCallback((focusMode: FocusMode) => {
        putActiveFocusMode({
            __typename: "ActiveFocusMode" as const,
            focusMode: {
                ...focusMode,
                __typename: "ActiveFocusModeFocusMode" as const,
            },
            stopCondition: FocusModeStopCondition.NextBegins,
        }, session);
    }, [putActiveFocusMode, session]);
    const handleDeactivate = useCallback(() => {
        putActiveFocusMode(null, session);
    }, [putActiveFocusMode, session]);

    // Handle dialog for adding new focus mode
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const [editingFocusMode, setEditingFocusMode] = useState<FocusMode | null>(null);
    function handleAdd() {
        if (!canAdd) {
            // If you don't have premium, open premium page
            if (!hasPremium) {
                setLocation(LINKS.Pro);
                PubSub.get().publish("snack", { message: "Upgrade to increase limit", severity: "Info" });
            }
            // Otherwise, show error message
            else PubSub.get().publish("snack", { message: "Max reached", severity: "Error" });
            return;
        }
        setIsDialogOpen(true);
    }
    function handleUpdate(focusMode: FocusMode) {
        setEditingFocusMode(focusMode);
        setIsDialogOpen(true);
    }
    function handleCloseDialog() {
        setIsDialogOpen(false);
    }
    const handleCompleted = useCallback((focusMode: FocusMode) => {
        let updatedFocusModes: FocusMode[];
        // Check if focus mode is already in list (i.e. updated instead of created)
        const existingFocusMode = focusModeInfo.all.find((existingFocusMode) => existingFocusMode.id === focusMode.id);
        if (existingFocusMode) {
            updatedFocusModes = focusModeInfo.all.map((existingFocusMode) => existingFocusMode.id === focusMode.id ? focusMode : existingFocusMode);
            setFocusModes(updatedFocusModes);
        }
        // Otherwise, add to list
        else {
            updatedFocusModes = [...focusModeInfo.all, focusMode];
            setFocusModes(updatedFocusModes);
        }
        // Publish updated focus mode list. This should also update the active focus mode, if necessary.
        const currentUser = getCurrentUser(session);
        const users: SessionUser[] = [
            ...session!.users!.filter((user) => user.id !== currentUser.id),
            {
                ...currentUser,
                focusModes: updatedFocusModes,
            } as SessionUser,
        ];
        PubSub.get().publish("session", {
            __typename: "Session" as const,
            users,
            isLoggedIn: true,
        });
        // Close dialog
        setIsDialogOpen(false);
    }, [focusModeInfo.all, session, setFocusModes]);

    const canDelete = focusModeInfo.all.length > 1;

    const [showMoreTags, setShowMoreTags] = useState<any[]>([]);
    const handleShowMoreTagsUpdate = useCallback(function handleShowMoreTagsUpdateCallback(tags: any[]) {
        setShowMoreTags(tags);
    }, []);
    useEffect(function updateActiveFocusModeState() {
        const fullInfo = focusModeInfo.all.find((focusMode) => focusMode.id === focusModeInfo.active?.focusMode?.id);
        if (fullInfo) {
            const showMoreFilters = fullInfo.filters?.filter((filter) => filter.filterType === FocusModeFilterType.ShowMore) ?? [];
            setShowMoreTags(showMoreFilters);
        } else {
            setShowMoreTags([]);
        }
    }, [focusModeInfo.active, focusModeInfo.all]);

    return (
        <ScrollBox>
            <FocusModeUpsert
                display="dialog"
                isCreate={editingFocusMode === null}
                isOpen={isDialogOpen}
                onCancel={handleCloseDialog}
                onClose={handleCloseDialog}
                onCompleted={handleCompleted}
                onDeleted={noop}
                overrideObject={editingFocusMode ?? fallbackOverrideObject}
            />
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("FocusMode", { count: 2 })}
            />
            <SettingsContent>
                <SettingsList />
                <Box m="auto" mt={2}>
                    <TitleStack>
                        <IconCommon
                            fill="background.textPrimary"
                            name="FocusMode"
                            size={40}
                            style={focusModeIconStyle}
                        />
                        <Typography component="h2" variant="h4">{t("FocusMode", { count: 2 })}</Typography>
                        <Tooltip title={canAdd ? "Add new" : "Max focus modes reached. Upgrade to premium to add more, or edit/delete an existing focus mode."} placement="top">
                            <AddIconButton
                                size="medium"
                                onClick={handleAdd}
                            >
                                <IconCommon
                                    fill={canAdd ? "secondary.main" : "grey.500"}
                                    name="Add"
                                />
                            </AddIconButton>
                        </Tooltip>
                    </TitleStack>
                    <ListContainer
                        emptyText={t("NoFocusModes", { ns: "error" })}
                        isEmpty={focusModeInfo.all.length === 0}
                        sx={listContainerStyle}
                    >
                        {focusModeInfo.all.map((focusMode) => (
                            <FocusModeListItem
                                key={focusMode.id}
                                focusMode={focusMode}
                                isActive={focusMode.id === focusModeInfo.active?.focusMode?.id}
                                canDelete={canDelete}
                                onActivate={handleActivate}
                                onDeactivate={handleDeactivate}
                                onDelete={handleDelete}
                                onEdit={handleUpdate}
                            />
                        ))}
                    </ListContainer>
                    <StyledDivider />
                    {!focusModeInfo.active ? (
                        <Typography variant="h6" align="center">
                            No focus mode selected.
                        </Typography>
                    ) : (
                        <Box>
                            <Title
                                addSidePadding={false}
                                title={getDisplay(focusModeInfo.active).title}
                                variant="subheader"
                            />
                            <FormSection variant="card">
                                <Typography variant="body1">
                                    Use the tag selector below to pick topics that you&apos;d like to see more when in this focus mode.
                                </Typography>
                                <WarningBox>
                                    <IconCommon name="Warning" style={warningIconStyle} />
                                    <Typography variant="body2">In the future, this will be used to customize the upcoming <i>Explore</i> page.</Typography>
                                </WarningBox>
                                <TagSelectorBase handleTagsUpdate={handleShowMoreTagsUpdate} tags={showMoreTags} />
                            </FormSection>
                            {/* Disabled update button */}
                            <Button
                                variant="contained"
                                disabled
                                fullWidth
                                onClick={noop}
                                startIcon={<IconCommon name="Save" />}
                                sx={updateActiveFocusModeButtonStyle}
                            >
                                Update
                            </Button>
                            <WarningBox>
                                <IconCommon name="Warning" style={warningIconStyle} />
                                <Typography variant="body2">This feature is not yet implemented.</Typography>
                            </WarningBox>
                        </Box>
                    )}
                </Box>
            </SettingsContent>
        </ScrollBox>
    );
}

type FocusModeListItemProps = {
    focusMode: FocusMode;
    isActive: boolean;
    canDelete: boolean;
    onActivate: (focusMode: FocusMode) => unknown;
    onDeactivate: (focusMode: FocusMode) => unknown;
    onDelete: (focusMode: FocusMode) => unknown;
    onEdit: (focusMode: FocusMode) => unknown;
}

const FocusModeListItem = memo(function FocusModeListItem({
    focusMode,
    isActive,
    canDelete,
    onActivate,
    onDeactivate,
    onDelete,
    onEdit,
}: FocusModeListItemProps) {
    const { palette } = useTheme();

    const handleEdit = useCallback(() => {
        onEdit(focusMode);
    }, [onEdit, focusMode]);

    const handleDeleteClick = useCallback((event: React.MouseEvent) => {
        event.stopPropagation();
        event.preventDefault();
        onDelete(focusMode);
    }, [onDelete, focusMode]);

    const handleToggleActive = useCallback((event: React.MouseEvent) => {
        event.stopPropagation();
        event.preventDefault();
        if (isActive) {
            onDeactivate(focusMode);
        } else {
            onActivate(focusMode);
        }
    }, [isActive, onActivate, onDeactivate, focusMode]);

    return (
        <StyledFocusModeListItem
            isActive={isActive}
            onClick={handleEdit}
        >
            <FocusModeListItemInner>
                <FocusModeNameContainer>
                    <FocusModeName variant="h6">{focusMode.name}</FocusModeName>
                    <IsActiveChip
                        label={isActive ? "Active" : "Inactive"}
                        size="small"
                        color="primary"
                        variant={isActive ? "filled" : "outlined"}
                        onClick={handleToggleActive}
                    />
                </FocusModeNameContainer>
                <FocusModeDescription primary={focusMode.description} />
            </FocusModeListItemInner>
            {canDelete && (
                <DeleteFocusModeBox
                    component="a"
                    onClick={handleDeleteClick}
                >
                    <IconCommon name="Delete" fill="secondary.main" />
                </DeleteFocusModeBox>
            )}
        </StyledFocusModeListItem>
    );
});
