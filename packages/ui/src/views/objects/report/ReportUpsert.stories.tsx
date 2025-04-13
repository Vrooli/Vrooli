/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { Report, ReportFor, endpointsReport, getObjectUrl, uuid } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ReportUpsert } from "./ReportUpsert.js";

// Create simplified mock data for Report responses
const mockReportData: Report = {
    __typename: "Report" as const,
    id: uuid(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    reason: "Inappropriate",
    otherReason: "",
    details: "This is a detailed explanation of why this content was reported as inappropriate. The content contains material that violates community guidelines.",
    language: "en",
    createdFor: {
        __typename: "StandardVersion" as const,
        id: uuid(),
    },
    you: {
        __typename: "ReportYou",
        canDelete: true,
        canRead: true,
        canUpdate: true,
    },
};

// Mock for other reason type report
const mockOtherReasonReportData: Report = {
    ...mockReportData,
    id: uuid(),
    reason: "Other",
    otherReason: "The content is misleading",
    details: "This content provides incorrect information that could mislead users.",
};

// Create mock for report object for a different object type
const mockForRoutineReportData: Report = {
    ...mockReportData,
    id: uuid(),
    createdFor: {
        __typename: "RoutineVersion" as const,
        id: uuid(),
    },
};

export default {
    title: "Views/Objects/Report/ReportUpsert",
    component: ReportUpsert,
};

// Create a new Report in a dialog
export function CreateDialog() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            createdFor={{
                __typename: "StandardVersion" as ReportFor,
                id: uuid(),
            }}
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

// Create a report for a Routine
export function CreateForRoutine() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={true}
            createdFor={{
                __typename: "RoutineVersion" as ReportFor,
                id: uuid(),
            }}
        />
    );
}
CreateForRoutine.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing Report in a dialog
export function UpdateDialog() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={false}
            isOpen={true}
            createdFor={mockReportData.createdFor as { __typename: ReportFor, id: string }}
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
            http.get(`${API_URL}/v2/rest${endpointsReport.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockReportData)}/edit`,
    },
};

// Update a report with "Other" reason
export function UpdateOtherReason() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={false}
            createdFor={mockOtherReasonReportData.createdFor as { __typename: ReportFor, id: string }}
        />
    );
}
UpdateOtherReason.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsReport.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockOtherReasonReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockOtherReasonReportData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={false}
            createdFor={mockReportData.createdFor as { __typename: ReportFor, id: string }}
        />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsReport.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockReportData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={true}
            createdFor={{
                __typename: "StandardVersion" as ReportFor,
                id: uuid(),
            }}
        />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <ReportUpsert
            display="dialog"
            isCreate={true}
            isOpen={true}
            createdFor={mockReportData.createdFor as { __typename: ReportFor, id: string }}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockReportData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
