import { NotePage, NoteVersion, NoteVersionTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

export const notePage: ApiPartial<NotePage> = {
    common: {
        id: true,
        pageIndex: true,
        text: true,
    },
};

export const noteVersionTranslation: ApiPartial<NoteVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
        pages: () => rel(notePage, "list"),
    },
};

export const noteVersion: ApiPartial<NoteVersion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        versionIndex: true,
        versionLabel: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./note")).note, "full", { omit: "versions" }),
        translations: () => rel(noteVersionTranslation, "full"),
        versionNotes: true,
    },
    list: {
        root: async () => rel((await import("./note")).note, "list", { omit: "versions" }),
        translations: () => rel(noteVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./note")).note, "nav", { omit: "versions" }),
        translations: () => rel(noteVersionTranslation, "list"),
    },
};
