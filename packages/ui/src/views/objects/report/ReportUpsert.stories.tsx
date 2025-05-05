/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { Report, ReportFor, endpointsReport, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ReportUpsert } from "./ReportUpsert.js";

// Create simplified mock data for Report responses
const mockReportData: Report = {
    __typename: "Report" as const,
    id: generatePKString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    reason: "Inappropriate",
    otherReason: "",
    details: "This is a detailed explanation of why this content was reported as inappropriate. The content contains material that violates community guidelines.",
    language: "en",
    createdFor: {
        __typename: "StandardVersion" as const,
        id: generatePKString(),
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
    id: generatePKString(),
    reason: "Other",
    otherReason: "The content is misleading",
    details: "This content provides incorrect information that could mislead users.",
};

// Create mock for report object for a different object type
const mockForRoutineReportData: Report = {
    ...mockReportData,
    id: generatePKString(),
    createdFor: {
        __typename: "RoutineVersion" as const,
        id: generatePKString(),
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
            display="Dialog"
            isCreate={true}
            isOpen={true}
            createdFor={{
                __typename: "StandardVersion" as ReportFor,
                id: generatePKString(),
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
            display="Dialog"
            isCreate={true}
            createdFor={{
                __typename: "RoutineVersion" as ReportFor,
                id: generatePKString(),
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
            display="Dialog"
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
            http.get(`${API_URL}/v2${endpointsReport.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockReportData)}/edit`,
    },
};

// Update a report with "Other" reason
export function UpdateOtherReason() {
    return (
        <ReportUpsert
            display="Dialog"
            isCreate={false}
            createdFor={mockOtherReasonReportData.createdFor as { __typename: ReportFor, id: string }}
        />
    );
}
UpdateOtherReason.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsReport.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockOtherReasonReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockOtherReasonReportData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <ReportUpsert
            display="Dialog"
            isCreate={false}
            createdFor={mockReportData.createdFor as { __typename: ReportFor, id: string }}
        />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2${endpointsReport.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2${getObjectUrl(mockReportData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <ReportUpsert
            display="Dialog"
            isCreate={true}
            createdFor={{
                __typename: "StandardVersion" as ReportFor,
                id: generatePKString(),
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
            display="Dialog"
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
