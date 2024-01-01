import i18next from "i18next";
// import { PubSub } from "../utils/pubsub"; TODO pubsub mock not working. Likely due to being a singleton class
import { errorToCode, errorToMessage, hasErrorCode } from "./errorParser";

jest.mock("i18next");
// jest.mock("../utils/pubsub", () => ({
//     PubSub: {
//         get: jest.fn().mockReturnValue({
//             publish: jest.fn(),
//         }),
//     },
// }));

describe("errorToCode", () => {
    beforeEach(() => { jest.clearAllMocks(); });

    it("should return the first error code", () => {
        const response = { errors: [{ code: "InputEmpty" }, { code: "FailedToDelete" }] };
        expect(errorToCode(response)).toEqual("InputEmpty");
    });

    it("should return \"ErrorUnknown\" if no error code is found", () => {
        const response = { errors: [{ message: "Error" }] };
        expect(errorToCode(response)).toEqual("ErrorUnknown");
    });
});

describe("errorToMessage", () => {
    beforeEach(() => { jest.clearAllMocks(); });

    it("should return the first error message", () => {
        const response = { errors: [{ message: "NotFound" }, { message: "FailedToCreate" }] };
        const languages = ["en"];
        expect(errorToMessage(response, languages)).toEqual("NotFound");
    });

    it("should return translated error code if no error message is found", () => {
        const response = { errors: [{ code: "NotFound" }] };
        const languages = ["en"];
        errorToMessage(response, languages);
        expect(i18next.t).toHaveBeenCalledWith("NotFound", expect.any(Object));
    });
});

describe("hasErrorCode", () => {
    beforeEach(() => { jest.clearAllMocks(); });

    it("should return true if the error code exists in the response", () => {
        const response = { errors: [{ code: "HardLockout" }] };
        expect(hasErrorCode(response, "HardLockout")).toBeTruthy();
    });

    it("should return false if the error code does not exist in the response", () => {
        const response = { errors: [{ code: "LineBreaksBio" }] };
        expect(hasErrorCode(response, "InternalError")).toBeFalsy();
    });
});

// describe("displayServerErrors", () => {
//     let pubsub;
//     beforeAll(async () => {
//         const module = await import("../utils/pubsub");
//         console.log("got module", module.PubSub.get());
//         pubsub = module.PubSub.get();
//     });

//     beforeEach(() => { jest.clearAllMocks(); });


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
