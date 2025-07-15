/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { endpointsReport, generatePK, getObjectUrl, type Report, type ReportFor } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockUrl, getStoryRouteEditPath } from "../../../__test/helpers/storybookMocking.js";
import { ReportUpsert } from "./ReportUpsert.js";

// Create simplified mock data for Report responses
const mockReportData: Report = {
    __typename: "Report" as const,
    id: generatePK().toString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    reason: "Inappropriate",
    otherReason: "",
    details: "This is a detailed explanation of why this content was reported as inappropriate. The content contains material that violates community guidelines.",
    language: "en",
    createdFor: {
        __typename: "StandardVersion" as const,
        id: generatePK().toString(),
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
    id: generatePK().toString(),
    reason: "Other",
    otherReason: "The content is misleading",
    details: "This content provides incorrect information that could mislead users.",
};

// Create mock for report object for a different object type
const mockForRoutineReportData: Report = {
    ...mockReportData,
    id: generatePK().toString(),
    createdFor: {
        __typename: "RoutineVersion" as const,
        id: generatePK().toString(),
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
                id: generatePK().toString(),
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
                id: generatePK().toString(),
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
            http.get(getMockUrl(endpointsReport.findOne), () => {
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockReportData),
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
            http.get(getMockUrl(endpointsReport.findOne), () => {
                return HttpResponse.json({ data: mockOtherReasonReportData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockOtherReasonReportData),
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
            http.get(getMockUrl(endpointsReport.findOne), async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockReportData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockReportData),
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
                id: generatePK().toString(),
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
