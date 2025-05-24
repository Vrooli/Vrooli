/* eslint-disable no-magic-numbers */
import { generatePK, type Reminder, type ReminderList as ReminderListType } from "@local/shared";
import { Box, Button, Typography } from "@mui/material";
import type { Meta } from "@storybook/react";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { ReminderList } from "./ReminderList.js";

const meta = {
    title: "Components/Lists/ReminderList",
    component: ReminderList,
    parameters: {
        docs: {
            description: {
                component: "A list component for displaying and managing reminders.",
            },
        },
    },
} satisfies Meta<typeof ReminderList>;

export default meta;

// Mock data for reminders
const mockReminderListId = generatePK().toString();
const mockReminderList: ReminderListType = {
    __typename: "ReminderList",
    id: mockReminderListId,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    reminders: [],
};

// Create mock reminders
const baseMockReminders: Reminder[] = [
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Complete weekly report",
        description: "Finalize and submit the weekly progress report",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 1)),
        isComplete: false,
        index: 0,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Meeting preparation",
        description: "Prepare slides for the upcoming team meeting",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 2)),
        isComplete: false,
        index: 1,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Review pull requests",
        description: "Review pending pull requests from the team",
        dueDate: new Date(new Date().setHours(new Date().getHours() + 4)),
        isComplete: true,
        index: 2,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Project planning",
        description: "Plan the next sprint objectives and tasks",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 3)),
        isComplete: false,
        index: 3,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Client follow-up",
        description: "Send follow-up emails to clients",
        dueDate: new Date(new Date().setHours(new Date().getHours() + 2)),
        isComplete: false,
        index: 4,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
];

// Extended mock reminders for the "long list" option
const extendedMockReminders: Reminder[] = [
    ...baseMockReminders,
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Code review",
        description: "Review code changes for the new feature",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 4)),
        isComplete: false,
        index: 5,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Documentation update",
        description: "Update documentation for the recent API changes",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 5)),
        isComplete: false,
        index: 6,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Team feedback session",
        description: "Conduct feedback session with team members",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 7)),
        isComplete: false,
        index: 7,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "Quarterly planning",
        description: "Prepare quarterly objectives and key results",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 10)),
        isComplete: false,
        index: 8,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        __typename: "Reminder",
        id: generatePK().toString(),
        name: "System maintenance",
        description: "Schedule and perform routine system maintenance",
        dueDate: new Date(new Date().setDate(new Date().getDate() + 14)),
        isComplete: false,
        index: 9,
        reminderItems: [],
        reminderList: mockReminderList,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
];

// Update the mock reminder list with reminders
const mockReminderListWithItems: ReminderListType = {
    ...mockReminderList,
    reminders: baseMockReminders,
};

const controlsContainerStyle = {
    marginBottom: 4,
    padding: 2,
    border: 1,
    borderColor: "divider",
    borderRadius: 1,
} as const;

const controlsRowStyle = {
    display: "flex",
    gap: 2,
    alignItems: "center",
    marginBottom: 2,
} as const;

const controlLabelStyle = {
    minWidth: 120,
} as const;

/**
 * Interactive story showcasing ReminderList with configurable options
 */
export function Interactive() {
    const [canUpdate, setCanUpdate] = useState(true);
    const [loading, setLoading] = useState(false);
    const [listState, setListState] = useState<"short" | "long" | "empty">("short");
    const [reminderListData, setReminderListData] = useState<ReminderListType>(mockReminderListWithItems);

    function handleUpdate(updatedList: ReminderListType) {
        setReminderListData(updatedList);
    }

    function handleCanUpdateClick() {
        setCanUpdate(!canUpdate);
    }

    function handleLoadingClick() {
        setLoading(!loading);
    }

    function handleListLengthClick() {
        // Cycle through states: short -> long -> empty -> short
        const nextState = listState === "short" ? "long" : listState === "long" ? "empty" : "short";
        setListState(nextState);

        const updatedReminderList = {
            ...mockReminderList,
            reminders: nextState === "long" ? extendedMockReminders :
                nextState === "short" ? baseMockReminders :
                    [],
        };

        setReminderListData(updatedReminderList);
    }

    return (
        <PageContainer>
            <ScrollBox>
                {/* Controls */}
                <Box sx={controlsContainerStyle}>
                    <Typography variant="h6" gutterBottom>Controls</Typography>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>List Length:</Typography>
                        <Button
                            variant="contained"
                            onClick={handleListLengthClick}
                            color={listState === "empty" ? "error" : "primary"}
                        >
                            {listState === "long" ? "Long List" :
                                listState === "short" ? "Short List" :
                                    "Empty List"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Can Update:</Typography>
                        <Button
                            variant={canUpdate ? "contained" : "outlined"}
                            onClick={handleCanUpdateClick}
                        >
                            {canUpdate ? "Enabled" : "Disabled"}
                        </Button>
                    </Box>

                    <Box sx={controlsRowStyle}>
                        <Typography sx={controlLabelStyle}>Loading:</Typography>
                        <Button
                            variant={loading ? "contained" : "outlined"}
                            onClick={handleLoadingClick}
                        >
                            {loading ? "Loading" : "Loaded"}
                        </Button>
                    </Box>
                </Box>

                {/* ReminderList */}
                <ReminderList
                    canUpdate={canUpdate}
                    loading={loading}
                    list={loading ? null : reminderListData}
                    handleUpdate={handleUpdate}
                    parent={{ id: "test" }}
                />
            </ScrollBox>
        </PageContainer>
    );
}

Interactive.parameters = {
    docs: {
        description: {
            story: "An interactive demo of the ReminderList component with configurable list length, update permissions, and loading states.",
        },
    },
}; 
