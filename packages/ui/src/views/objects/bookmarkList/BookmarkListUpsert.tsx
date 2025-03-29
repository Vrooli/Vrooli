import { Bookmark, BookmarkList, BookmarkListCreateInput, BookmarkListShape, BookmarkListUpdateInput, bookmarkListValidation, BookmarkShape, DeleteOneInput, DeleteType, DUMMY_ID, endpointsActions, endpointsBookmarkList, ListObject, noopSubmit, Session, shapeBookmarkList, Success, uuid } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper, useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../../components/containers/ListContainer.js";
import { FindObjectDialog } from "../../../components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { ObjectListActions } from "../../../components/lists/types.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { EditableTitle } from "../../../components/text/EditableTitle.js";
import { SessionContext } from "../../../contexts.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useBulkObjectActions, useObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { ActionIcon, AddIcon, CancelIcon, DeleteIcon } from "../../../icons/common.js";
import { useLocation } from "../../../route/router.js";
import { FormContainer } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { BulkObjectAction } from "../../../utils/actions/bulkObjectActions.js";
import { DUMMY_LIST_LENGTH } from "../../../utils/consts.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { BookmarkListFormProps, BookmarkListUpsertProps } from "./types.js";

function bookmarkListInitialValues(
    session: Session | undefined,
    existing?: Partial<BookmarkList> | null | undefined,
): BookmarkListShape {
    return {
        __typename: "BookmarkList" as const,
        id: DUMMY_ID,
        label: "Bookmark List",
        bookmarks: [],
        ...existing,
    };
}

function transformBookmarkListValues(values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) {
    return isCreate ? shapeBookmarkList.create(values) : shapeBookmarkList.update(existing, values);
}

function BookmarkListForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: BookmarkListFormProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [bookmarksField, , bookmarksHelpers] = useField<BookmarkShape[]>("bookmarks");

    const addNewBookmark = useCallback((to: BookmarkShape) => {
        bookmarksHelpers.setValue([...bookmarksField.value, {
            __typename: "Bookmark" as const,
            id: uuid(),
            to,
            list: { __typename: "BookmarkList", id: values.id },
        } as any]);
    }, [bookmarksField.value, bookmarksHelpers, values.id]);

    // Search dialog to find objects to bookmark
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedObject?: any) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedObject;
        if (selectedObject) {
            addNewBookmark(selectedObject);
        }
    }, [addNewBookmark]);

    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<BookmarkList>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "BookmarkList",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsBookmarkList.createOne,
        endpointUpdate: endpointsBookmarkList.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "BookmarkList" });

    const onSubmit = useSubmitHelper<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>({
        disabled,
        existing,
        fetch,
        inputs: transformBookmarkListValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const actionData = useObjectActions({
        object: existing as ListObject,
        objectType: "BookmarkList",
        setLocation,
        setObject: handleUpdate,
    });

    // Handle delete
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback(() => {
        function performDelete() {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteMutation,
                inputs: { id: values.id, objectType: DeleteType.BookmarkList },
                successCondition: (data) => data.success,
                successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values as ListObject).title ?? t("BookmarkList", { count: 1 }) } }),
                onSuccess: () => { handleDeleted(values as BookmarkList); },
                errorMessage: () => ({ messageKey: "FailedToDelete" }),
            });
        }
        PubSub.get().publish("alertDialog", {
            messageKey: "DeleteConfirm",
            buttons: [{
                labelKey: "Delete",
                onClick: performDelete,
            }, {
                labelKey: "Cancel",
            }],
        });
    }, [deleteMutation, values, t, handleDeleted]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<BookmarkShape>(bookmarksField.value);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<BookmarkShape>({
        allData: bookmarksField.value,
        selectedData,
        setAllData: bookmarksHelpers.setValue,
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });

    const onAction = useCallback((action: keyof ObjectListActions<Bookmark>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const deletedId = data[0] as ArgsType<ObjectListActions<Bookmark>["Deleted"]>[0];
                const filteredBookmarks = bookmarksField.value.filter((bookmark) => bookmark.id !== deletedId);
                bookmarksHelpers.setValue([...filteredBookmarks]);
                break;
            }
            case "Updated": {
                const updatedBookmark = data[0] as ArgsType<ObjectListActions<Bookmark>["Updated"]>[0];
                const index = bookmarksField.value.findIndex((bookmark) => bookmark.id === updatedBookmark.id);
                if (index !== -1) return;
                const newBookmarks = [...bookmarksField.value];
                newBookmarks[index] = updatedBookmark;
                bookmarksHelpers.setValue(newBookmarks);
                break;
            }
        }
    }, [bookmarksField.value, bookmarksHelpers]);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || isDeleteLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, isDeleteLoading, props.isSubmitting]);

    const actionIconProps = useMemo(() => ({ fill: palette.secondary.contrastText, width: "36px", height: "36px" }), [palette.secondary.contrastText]);

    const sideActionButtons = useMemo(() => {
        const buttons: JSX.Element[] = [];
        if (isSelecting && selectedData.length) {
            buttons.push(
                <Tooltip title={t("Delete")}>
                    <IconButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }} sx={{ background: palette.secondary.main }}>
                        <DeleteIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip>,
            );
        }
        buttons.push(
            <Tooltip title={t(isSelecting ? "Cancel" : "Select")}>
                <IconButton aria-label={t(isSelecting ? "Cancel" : "Select")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                    {isSelecting ? <CancelIcon {...actionIconProps} /> : <ActionIcon {...actionIconProps} />}
                </IconButton>
            </Tooltip>,
        );
        if (!isSelecting) {
            buttons.push(
                <Tooltip title={"Add Bookmark"}>
                    <IconButton aria-label={"Add new bookmark"} onClick={openSearch} sx={{ background: palette.secondary.main }}>
                        <AddIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip>,
            );
        }
        if (buttons.length === 0) {
            return null;
        }
        return buttons;
    }, [actionIconProps, handleToggleSelecting, isSelecting, onBulkActionStart, openSearch, palette.secondary.main, selectedData.length, t]);


    return (
        <MaybeLargeDialog
            display={display}
            id="bookmark-list-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            {BulkDeleteDialogComponent}
            <TopBar
                display={display}
                onClose={onClose}
                titleComponent={<EditableTitle
                    handleDelete={handleDelete}
                    isDeletable={!(isCreate || disabled)}
                    isEditable={!disabled}
                    titleField="label"
                    variant="subheader"
                    sxs={{
                        stack: {
                            padding: 0,
                            ...(display === "page" && !isMobile ? {
                                margin: "auto",
                                maxWidth: "800px",
                                paddingTop: 1,
                                paddingBottom: 1,
                            } : {}),
                        },
                    }}
                    DialogContentForm={() => (
                        <BaseForm
                            display="dialog"
                            style={{
                                paddingBottom: "16px",
                            }}
                        >
                            <FormContainer>
                                <Field
                                    fullWidth
                                    name="label"
                                    label={t("Label")}
                                    as={TextInput}
                                />
                            </FormContainer>
                        </BaseForm>
                    )}
                />}
            />
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                style={{
                    width: "min(700px, 100vw)",
                    flex: 1,
                    margin: "unset",
                    marginLeft: "auto",
                    marginRight: "auto",
                }}
            >
                <ListContainer
                    emptyText={t("NoResults", { ns: "error" })}
                    isEmpty={bookmarksField.value.length === 0 && !isLoading}
                >
                    <ObjectList
                        dummyItems={new Array(DUMMY_LIST_LENGTH).fill("BookmarkList")}
                        handleToggleSelect={handleToggleSelect}
                        isSelecting={isSelecting}
                        items={bookmarksField.value as ListObject[]}
                        keyPrefix={"bookmark-item"}
                        loading={isLoading}
                        onAction={onAction}
                        selectedItems={selectedData}
                    />
                </ListContainer>
            </BaseForm>
            <Box sx={{
                position: "absolute",
                left: 0,
                right: 0,
                bottom: 0,
            }}>
                <BottomActionsButtons
                    display={display}
                    errors={props.errors}
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={handleCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={onSubmit}
                    sideActionButtons={sideActionButtons}
                />
            </Box>
        </MaybeLargeDialog>
    );
}

export function BookmarkListUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BookmarkListUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useManagedObject<BookmarkList, BookmarkListShape>({
        ...endpointsBookmarkList.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "BookmarkList",
        overrideObject,
        transform: (data) => bookmarkListInitialValues(session, data),
    });

    async function validateValues(values: BookmarkListShape) {
        return await validateFormValues(values, existing, isCreate, transformBookmarkListValues, bookmarkListValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <BookmarkListForm
                disabled={false}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
