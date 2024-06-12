import { Bookmark, BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, bookmarkListValidation, DeleteOneInput, DeleteType, DUMMY_ID, endpointGetBookmarkList, endpointPostBookmarkList, endpointPostDeleteOne, endpointPutBookmarkList, ListObject, noopSubmit, Session, Success, uuid } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper, useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useSelectableList } from "hooks/useSelectableList";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { ActionIcon, AddIcon, CancelIcon, DeleteIcon } from "icons";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer } from "styles";
import { ArgsType } from "types";
import { BulkObjectAction } from "utils/actions/bulkObjectActions";
import { getDisplay } from "utils/display/listTools";
import { PubSub } from "utils/pubsub";
import { BookmarkShape } from "utils/shape/models/bookmark";
import { BookmarkListShape, shapeBookmarkList } from "utils/shape/models/bookmarkList";
import { validateFormValues } from "utils/validateFormValues";
import { BookmarkListFormProps, BookmarkListUpsertProps } from "../types";

const bookmarkListInitialValues = (
    session: Session | undefined,
    existing?: Partial<BookmarkList> | null | undefined,
): BookmarkListShape => ({
    __typename: "BookmarkList" as const,
    id: DUMMY_ID,
    label: "Bookmark List",
    bookmarks: [],
    ...existing,
});

const transformBookmarkListValues = (values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) =>
    isCreate ? shapeBookmarkList.create(values) : shapeBookmarkList.update(existing, values);

const BookmarkListForm = ({
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
}: BookmarkListFormProps) => {
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
        endpointCreate: endpointPostBookmarkList,
        endpointUpdate: endpointPutBookmarkList,
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
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback(() => {
        const performDelete = () => {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteMutation,
                inputs: { id: values.id, objectType: DeleteType.BookmarkList },
                successCondition: (data) => data.success,
                successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values as ListObject).title ?? t("BookmarkList", { count: 1 }) } }),
                onSuccess: () => { handleDeleted(values as BookmarkList); },
                errorMessage: () => ({ messageKey: "FailedToDelete" }),
            });
        };
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
    } = useSelectableList<BookmarkShape>();
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
    }, []);

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
    }, [actionData]);


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
                        <>
                            <BaseForm
                                display="dialog"
                                style={{
                                    width: "min(700px, 100vw)",
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
                        </>
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
                        dummyItems={new Array(5).fill("BookmarkList")}
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
};

export const BookmarkListUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: BookmarkListUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<BookmarkList, BookmarkListShape>({
        ...endpointGetBookmarkList,
        isCreate,
        objectType: "BookmarkList",
        overrideObject,
        transform: (data) => bookmarkListInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformBookmarkListValues, bookmarkListValidation)}
        >
            {(formik) => <BookmarkListForm
                disabled={false}
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
};
