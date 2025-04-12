export const BUSINESS_NAME = "Vrooli";
export const EMAIL = {
    Label: "info@vrooli.com",
    Link: "mailto:info@vrooli.com",
};
export const SUPPORT_EMAIL = {
    Label: "support@vrooli.com",
    Link: "mailto:support@vrooli.com",
};
export const SOCIALS = {
    Discord: "https://discord.gg/VyrDFzbmmF",
    GitHub: "https://github.com/MattHalloran/Vrooli",
    X: "https://x.com/VrooliOfficial",
};
export const APP_URL = "https://vrooli.com";

/**
 * IDs for built-in (seeded) data
 */
export const SEEDED_IDS = {
    Code: {
        parseRunIOFromPlaintext: "00d91282-f1fa-4ddc-8164-9a48d0958c62",
        parseSearchTermsFromPlaintext: "77790e25-08c5-45be-bfd1-d2a5aa5f2515",
        listToPlaintext: "d53f6f5e-0330-4197-b326-b35996895cb5",
        listToNumberedPlaintext: "520bad8d-4b6e-4524-a9e5-cc592b1943fa",
    },
    CodeVersion: {
        parseRunIOFromPlaintext: "3d32ef12-bda9-42f5-8119-610314626f00",
        parseSearchTermsFromPlaintext: "b11f91fa-267c-49ee-8d30-9b8303041f9d",
        listToPlaintext: "3bab7464-0f49-475f-80d0-bffb93263c1c",
        listToNumberedPlaintext: "b7c51b22-9ec7-4ce7-87e5-b4f37afc1ea1",
    },
    Routine: {
        MintNft: "4e038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9",
        MintToken: "3f038f3b-f8f9-4f9b-8f9b-f8f9b8f9b8f9",
        ProjectKickoffChecklist: "9daf1edb-b98f-41f0-9d76-aab3539d671a",
        WorkoutPlanGenerator: "3daf1bdb-a98f-41f0-9d76-cab3539d671c",
    },
    Standard: {
        Cip0025: "3a038a3b-f8a9-4fab-8fab-c8a4baaab8d2",
    },
    Tag: {
        Ai: "1ea8e149-6588-43f4-8ebe-4d599a357976",
        Automation: "d508b046-bccc-45f0-bcbf-05640dec7ffd",
        Cardano: "bd555cf8-b2ff-4df7-89c0-2e9783fe71ff",
        Cip: "39814a12-defa-4378-b8c9-e804566983b4",
        Collaboration: "baae3417-e4b6-4b68-ad4c-91a03098f007",
        Entrepreneurship: "9e6488dc-d235-4a78-8313-3ba8a0abde8d",
        Vrooli: "ef6cc592-3d5f-4bb0-98ac-f428a484975c",
    },
    Team: {
        Vrooli: "60aea87a-2e75-48ee-bf2f-f0a48db00d4a",
    },
    User: {
        Admin: "3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2",
        Valyxa: "4b038f3b-f1f7-1f9b-8f4b-cff4b8f9b20f",
    },
} as const;

export const SEEDED_TAGS = {
    Ai: "Artificial Intelligence (AI)",
    Automation: "Automation",
    Cardano: "Cardano",
    Cip: "Cardano Improvement Proposal (CIP)",
    Collaboration: "Collaboration",
    Entrepreneurship: "Entrepreneurship",
    Vrooli: "Vrooli",
} as const;
