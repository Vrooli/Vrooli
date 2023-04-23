import { rel } from "../utils";
export const chatParticipant = {
    __typename: "ChatParticipant",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
    },
    list: {
        user: async () => rel((await import("./user")).user, "nav"),
    },
};
//# sourceMappingURL=chatParticipant.js.map