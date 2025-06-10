import i18next from "i18next";
// import { PubSub } from "../utils/pubsub.js"; TODO pubsub mock not working. Likely due to being a singleton class
import { type ServerResponse } from "@vrooli/shared";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { ServerResponseParser } from "./responseParser.js";

vi.mock("i18next", () => ({
    default: {
        t: vi.fn((key) => key),
    },
}));
// vi.mock("../utils/pubsub", () => ({
//     PubSub: {
//         get: vi.fn().mockReturnValue({
//             publish: vi.fn(),
//         }),
//     },
// }));

describe("ServerResponseParser", () => {
    describe("errorToCode", () => {
        beforeEach(() => { vi.clearAllMocks(); });

        it("should return the first error code", () => {
            const response = {
                errors: [
                    { code: "InputEmpty", trace: "0000-asdf" },
                    { code: "FailedToDelete", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            expect(ServerResponseParser.errorToCode(response)).toEqual("InputEmpty");
        });

        it("should return \"ErrorUnknown\" if no error code is found", () => {
            const response = {
                errors: [
                    { message: "Error", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            expect(ServerResponseParser.errorToCode(response)).toEqual("ErrorUnknown");
        });
    });

    describe("errorToMessage", () => {
        beforeEach(() => { vi.clearAllMocks(); });

        it("should return the first error message", () => {
            const response = {
                errors: [
                    { message: "NotFound", trace: "0000-asdf" },
                    { message: "FailedToCreate", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            const languages = ["en"];
            expect(ServerResponseParser.errorToMessage(response, languages)).toEqual("NotFound");
        });

        it("should return translated error code if no error message is found", () => {
            const response = {
                errors: [
                    { code: "NotFound", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            const languages = ["en"];
            const result = ServerResponseParser.errorToMessage(response, languages);
            expect(i18next.t).toHaveBeenCalledWith("NotFound", { lng: "en", defaultValue: "Unknown error occurred." });
            expect(result).toEqual("NotFound"); // The mock returns the key
        });
    });

    describe("hasErrorCode", () => {
        beforeEach(() => { vi.clearAllMocks(); });

        it("should return true if the error code exists in the response", () => {
            const response = {
                errors: [
                    { code: "HardLockout", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            expect(ServerResponseParser.hasErrorCode(response, "HardLockout")).toBeTruthy();
        });

        it("should return false if the error code does not exist in the response", () => {
            const response = {
                errors: [
                    { code: "LineBreaksBio", trace: "0000-asdf" },
                ],
            } as ServerResponse;
            expect(ServerResponseParser.hasErrorCode(response, "InternalError")).toBeFalsy();
        });
    });

    // describe("displayServerErrors", () => {
    //     let pubsub;
    //     before(async () => {
    //         const module = await import("../utils/pubsub");
    //         console.log("got module", module.PubSub.get());
    //         pubsub = module.PubSub.get();
    //     });

    //     beforeEach(() => { vi.clearAllMocks(); });


    //     it("should display each error as a snack message", async () => {
    //         const errors = [{ message: "Error1" }, { message: "Error2" }];
    //         displayServerErrors(errors);
    //         expect(pubsub.publish).toHaveBeenCalledWith("snack", expect.anything());
    //         expect(pubsub.publish).toHaveBeenCalledTimes(2);
    //     });

    //     it("should not display anything if there are no errors", async () => {
    //         displayServerErrors();
    //         expect(pubsub.publish).not.toHaveBeenCalled();
    //     });
    // });
});
