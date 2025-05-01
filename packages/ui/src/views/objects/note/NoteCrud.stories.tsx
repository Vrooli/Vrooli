/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { DUMMY_ID, NoteVersion, Tag, User, endpointsNoteVersion, generatePKString, getObjectUrl } from "@local/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { NoteCrud } from "./NoteCrud.js";

// Create simplified mock data for Note responses
const mockNoteVersionData: NoteVersion = {
    __typename: "NoteVersion" as const,
    id: generatePKString(),
    comments: [],
    commentsCount: 0,
    createdAt: new Date().toISOString(),
    forks: [],
    forksCount: 0,
    isLatest: true,
    isPrivate: false,
    pullRequest: null,
    reports: [],
    reportsCount: 0,
    root: {
        __typename: "Note" as const,
        id: generatePKString(),
        isPrivate: false,
        owner: { __typename: "User" as const, id: generatePKString() } as User,
        parent: null,
        tags: Array.from({ length: 4 }, () => ({
            __typename: "Tag" as const,
            id: generatePKString(),
            tag: ["Notes", "Research", "Ideas", "Documentation", "Meeting Notes", "Project Notes"][Math.floor(Math.random() * 6)],
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        })) as Tag[],
    } as any,
    translations: [{
        __typename: "NoteVersionTranslation" as const,
        id: DUMMY_ID,
        language: "en",
        description: "This is a brief description of this note. It contains important information and ideas.",
        name: `Project Meeting Notes ${Math.floor(Math.random() * 100)}`,
        pages: [{
            __typename: "NotePage" as const,
            id: DUMMY_ID,
            pageIndex: 0,
            text: `# Project Meeting Notes
            
## Agenda
- Project status update
- Timeline review
- Resource allocation
- Next steps

## Discussion
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam auctor, nisl eget ultricies tincidunt, nisl nisl aliquam nisl, eget ultricies nisl nisl eget nisl. Nullam auctor, nisl eget ultricies tincidunt, nisl nisl aliquam nisl, eget ultricies nisl nisl eget nisl.

## Action Items
- [ ] Update project timeline
- [ ] Allocate resources for next sprint
- [ ] Schedule follow-up meeting
- [ ] Share meeting notes with stakeholders`,
        }],
    }],
    updatedAt: new Date().toISOString(),
    versionIndex: 1,
    versionLabel: `1.0.${Math.floor(Math.random() * 10)}`,
    versionNotes: "Initial version",
    you: {
        __typename: "VersionYou",
        canComment: true,
        canCopy: true,
        canDelete: true,
        canRead: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
    },
};

export default {
    title: "Views/Objects/Note/NoteCrud",
    component: NoteCrud,
};

// Create a new Note
export function Create() {
    return (
        <NoteCrud display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Note in a dialog
export function CreateDialog() {
    return (
        <NoteCrud
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

// Update an existing Note
export function Update() {
    return (
        <NoteCrud display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsNoteVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockNoteVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockNoteVersionData)}/edit`,
    },
};

// Update an existing Note in a dialog
export function UpdateDialog() {
    return (
        <NoteCrud
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
            http.get(`${API_URL}/v2/rest${endpointsNoteVersion.findOne.endpoint}`, () => {
                return HttpResponse.json({ data: mockNoteVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockNoteVersionData)}/edit`,
    },
};

// Loading state
export function Loading() {
    return (
        <NoteCrud display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsNoteVersion.findOne.endpoint}`, async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockNoteVersionData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockNoteVersionData)}/edit`,
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <NoteCrud display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// View mode (disabled)
export function ViewMode() {
    return (
        <NoteCrud display="Page" isCreate={false} />
    );
}
ViewMode.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(`${API_URL}/v2/rest${endpointsNoteVersion.findOne.endpoint}`, () => {
                const viewOnlyData = {
                    ...mockNoteVersionData,
                    you: {
                        ...mockNoteVersionData.you,
                        canUpdate: false,
                    },
                };
                return HttpResponse.json({ data: viewOnlyData });
            }),
        ],
    },
    route: {
        path: `${API_URL}/v2/rest${getObjectUrl(mockNoteVersionData)}`,
    },
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <NoteCrud
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockNoteVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
