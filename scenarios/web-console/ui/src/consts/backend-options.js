export const BACKEND_OPTIONS = [
    {
        id: "standard",
        label: "Standard",
        description: "Lightweight session. Lost if web console restarts.",
        survivesRestart: false,
    },
    {
        id: "persistent",
        label: "Persistent",
        description: "Survives restarts. Ideal for long-running tasks.",
        survivesRestart: true,
    },
];
