import { BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput, endpointGetBookmarkList, endpointPostBookmarkList, endpointPutBookmarkList } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BookmarkListForm, bookmarkListInitialValues, transformBookmarkListValues, validateBookmarkListValues } from "forms/BookmarkListForm/BookmarkListForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { BookmarkListShape } from "utils/shape/models/bookmarkList";
import { BookmarkListUpsertProps } from "../types";

export const BookmarkListUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: BookmarkListUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<BookmarkList, BookmarkListShape>({
        ...endpointGetBookmarkList,
        objectType: "BookmarkList",
        overrideObject,
        transform: (data) => bookmarkListInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<BookmarkList, BookmarkListCreateInput, BookmarkListUpdateInput>({
        display,
        endpointCreate: endpointPostBookmarkList,
        endpointUpdate: endpointPutBookmarkList,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="bookmark-list-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateBookmarkList" : "UpdateBookmarkList")}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<BookmarkListCreateInput | BookmarkListUpdateInput, BookmarkList>({
                        fetch,
                        inputs: transformBookmarkListValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateBookmarkListValues(values, existing, isCreate)}
            >
                {(formik) => <BookmarkListForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
