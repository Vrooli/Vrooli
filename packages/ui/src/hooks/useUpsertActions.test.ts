/* eslint-disable @typescript-eslint/ban-ts-comment */
import { GqlModelType, LINKS } from "@local/shared";
import { asMockObject, goBack } from "./useUpsertActions";

describe("goBack function", () => {
    const mockSetLocation = jest.fn();
    const mockHistoryBack = jest.spyOn(window.history, "back");

    beforeEach(() => {
        sessionStorage.clear();
        jest.clearAllMocks();
    });

    it("should navigate back if last path matches the target URL (ignoring query params)", () => {
        sessionStorage.setItem("lastPath", "/previous-page?query=123");
        goBack(mockSetLocation, "/previous-page?anotherQuery=456&query=123");
        expect(mockHistoryBack).toHaveBeenCalled();
        expect(mockSetLocation).not.toHaveBeenCalled();
    });

    it("should navigate back if there is a last path and no target URL provided", () => {
        sessionStorage.setItem("lastPath", "/previous-page");
        goBack(mockSetLocation);
        expect(mockHistoryBack).toHaveBeenCalled();
        expect(mockSetLocation).not.toHaveBeenCalled();
    });

    it("should navigate to the target URL if last path does not match the target URL", () => {
        sessionStorage.setItem("lastPath", "/some-other-page");
        goBack(mockSetLocation, "/target-page");
        expect(mockSetLocation).toHaveBeenCalledWith("/target-page", { replace: true });
        expect(mockHistoryBack).not.toHaveBeenCalled();
    });

    it("should navigate to the home link if no target URL provided and no last path", () => {
        goBack(mockSetLocation);
        expect(mockSetLocation).toHaveBeenCalledWith(LINKS.Home, { replace: true });
        expect(mockHistoryBack).not.toHaveBeenCalled();
    });

    it("should navigate to the target URL with replace: true when last path does not match target URL", () => {
        sessionStorage.setItem("lastPath", "/different-page");
        goBack(mockSetLocation, "/new-page");
        expect(mockSetLocation).toHaveBeenCalledWith("/new-page", { replace: true });
    });
});

describe("asMockObject", () => {
    it("should create a basic object without root", () => {
        const objectType = "Routine";
        const objectId = "123";
        const result = asMockObject(objectType, objectId);
        expect(result).toEqual({
            __typename: objectType,
            id: objectId,
        });
    });

    it("should create a versioned object with root", () => {
        const objectType = GqlModelType.CodeVersion;
        const objectId = "123";
        const rootObjectId = "456";
        const result = asMockObject(objectType, objectId, rootObjectId);
        expect(result).toEqual({
            __typename: objectType,
            id: objectId,
            root: {
                __typename: objectType.replace("Version", ""),
                id: rootObjectId,
            },
        });
    });

    it("should handle object type transformation correctly", () => {
        const objectType = "ApiVersion";
        const objectId = "789";
        const rootObjectId = "101";
        const result = asMockObject(objectType, objectId, rootObjectId);
        // @ts-ignore Testing runtime scenario
        expect(result.root.__typename).toBe(objectType.replace("Version", ""));
    });
});
