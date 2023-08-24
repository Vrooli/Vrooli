/**
 * Displays a list of emails for the user to manage
 */
import { endpointPutReminder, GqlModelType, LINKS, Reminder, ReminderUpdateInput } from "@local/shared";
import { List, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TitleContainer } from "components/containers/TitleContainer/TitleContainer";
import { useDisplayServerError } from "hooks/useDisplayServerError";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, OpenInNewIcon, ReminderIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { MyStuffPageTabOption } from "utils/search/objectToSearch";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderListItem } from "../ReminderListItem/ReminderListItem";
import { ReminderListProps } from "../types";

export const ReminderList = ({
    handleUpdate,
    id,
    listId,
    loading,
    reminders,
}: ReminderListProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Internal state
    const [allReminders, setAllReminders] = useState<Reminder[]>(reminders);
    useEffect(() => {
        setAllReminders(reminders);
    }, [reminders]);

    const handleUpdated = useCallback((index: number, reminder: Reminder) => {
        const newList = [...allReminders];
        newList[index] = reminder;
        setAllReminders(newList);
        handleUpdate && handleUpdate(newList);
    }, [allReminders, handleUpdate]);

    // Handle update mutation
    const [updateMutation, { errors: updateErrors }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointPutReminder);
    useDisplayServerError(updateErrors);
    const saveUpdate = useCallback((updated: Reminder) => {
        const index = allReminders.findIndex((reminder) => reminder.id === updated.id);
        if (index < 0) return;
        const original = allReminders[index];
        // Don't wait for the mutation to call handleUpdated
        handleUpdated(index, updated);
        // Call the mutation
        fetchLazyWrapper<ReminderUpdateInput, Reminder>({
            fetch: updateMutation,
            inputs: shapeReminder.update(original, updated),
            successCondition: (data) => !!data.id,
            successMessage: () => ({ messageKey: "ObjectUpdated", messageVariables: { objectName: updated.name } }),
        });
    }, [allReminders, handleUpdated, updateMutation]);

    const handleDeleted = useCallback((id: string) => {
        const index = allReminders.findIndex((reminder) => reminder.id === id);
        if (index < 0) return;
        const newList = [...allReminders];
        newList.splice(index, 1);
        setAllReminders(newList);
        handleUpdate && handleUpdate(newList);
    }, [allReminders, handleUpdate]);

    const onAction = useCallback((action: "Delete" | "Update", data: any) => {
        switch (action) {
            // case "Delete": //TODO
            //     handleDelete(data);
            //     break;
            case "Update":
                saveUpdate(data);
                break;
        }
    }, [saveUpdate]);

    return (
        <TitleContainer
            Icon={ReminderIcon}
            id={id}
            title={t("ToDo")}
            options={[{
                Icon: OpenInNewIcon,
                label: t("SeeAll"),
                onClick: () => { setLocation(`${LINKS.MyStuff}?type=${MyStuffPageTabOption.Reminder}`); },
            }, {
                Icon: AddIcon,
                label: t("Create"),
                onClick: () => { setLocation(`${LINKS.Reminder}/add`); },
            }]}
        >
            <>
                {/* Empty text */}
                {reminders.length === 0 && <Typography variant="h6" sx={{
                    textAlign: "center",
                    paddingTop: "8px",
                }}>{t("NoResults", { ns: "error" })}</Typography>}
                {/* Existing reminders */}
                <List>
                    {reminders.map((reminder, index) => (
                        <ReminderListItem
                            key={`reminder-${index}`}
                            data={reminder}
                            loading={false}
                            objectType={GqlModelType.Reminder}
                            onAction={onAction}
                        />
                    ))}
                </List>
            </>
        </TitleContainer>
    );
};
