/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { Reminder, endpointsReminder, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ReminderCrud } from "./ReminderCrud.js";

// Create simplified mock data for Reminder responses
const mockReminderData = {
    __typename: "Reminder" as const,
    id: generatePKString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    dueDate: new Date(Date.now() + 86400000).toISOString(), // Tomorrow
    index: 0,
    isComplete: false,
    name: "Complete project documentation",
    description: "This is a **detailed** description for the reminder using markdown.\nInclude all technical specifications and user guides.",
    reminderItems: Array.from({ length: 3 }, (_, i) => ({
        __typename: "ReminderItem" as const,
        id: generatePKString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        dueDate: new Date(Date.now() + (i + 1) * 43200000).toISOString(), // Staggered due dates
        index: i,
        isComplete: i === 0, // First item is complete
        name: ["Outline documentation structure", "Write technical sections", "Create user guide"][i],
        description: `Step ${i + 1} description with detailed instructions for completing this task.`,
    })),
    reminderList: {
        __typename: "ReminderList" as const,
        id: generatePKString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        reminders: [],
    },
} as unknown as Reminder;

export default {
    title: "Views/Objects/Reminder/ReminderCrud",
    component: ReminderCrud,
};

// Create a new Reminder
export function Create() {
    return (
        <ReminderCrud display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Reminder in a dialog
export function CreateDialog() {
    return (
        <ReminderCrud
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
CreateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing Reminder
export function Update() {
    return (
        <ReminderCrud display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsReminder.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockReminderData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockReminderData)}/edit`,
    },
};

// Update an existing Reminder in a dialog
export function UpdateDialog() {
    return (
        <ReminderCrud
            display="Dialog"
            isCreate={false}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
UpdateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsReminder.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockReminderData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockReminderData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ReminderCrud display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsReminder.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockReminderData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockReminderData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ReminderCrud display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <ReminderCrud
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockReminderData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
