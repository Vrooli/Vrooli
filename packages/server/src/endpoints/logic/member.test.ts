import { type FindByIdInput, type MemberSearchInput, type MemberUpdateInput } from "@local/shared";
import { expect } from "chai";
import sinon from "sinon";
import * as reads from "../../actions/reads.js";
import * as updates from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type Context as ApiContext } from "../../middleware/context.js";
import { member } from "./member.js"; // Adjust the import path as necessary

/**
 * Creates a mock context object for testing API endpoints.
 * @param permissions - Optional permissions to assign to the mock request.
 * @returns A mock ApiContext object.
 */
function createMockContext(): ApiContext {
    const mockReq = {
        // Mock request properties as needed, e.g., headers, user, etc.
        ip: "127.0.0.1",
        user: { id: "user-123" }, // Example user
    };

    // Create a mock object for RequestService methods
    const mockRequestServiceMethods = {
        rateLimit: sinon.stub().resolves(), // Stub rateLimit
        assertRequestFrom: sinon.stub().returns(undefined), // Stub assertRequestFrom
        // Add stubs for other RequestService methods if needed
    };

    // Stub the static get method to return our mock methods object
    // Adjust this if RequestService.get() returns an instance with these methods
    sinon.stub(RequestService, "get").returns(mockRequestServiceMethods as any);

    return {
        req: mockReq as any, // Cast as any to simplify mocking
        res: {} as any, // Add mock res object
        // Mock other context properties if needed, e.g., dataSources
    };
}

/**
 * Creates mock info object for endpoint tests.
 * In a real GraphQL scenario, this would contain query details.
 */
const mockInfo = {} as any; // Use a simple mock for info

describe("Member Endpoints", () => {
    let readOneStub: sinon.SinonStub;
    let readManyStub: sinon.SinonStub;
    let updateOneStub: sinon.SinonStub;
    let requestServiceMock: { rateLimit: sinon.SinonStub; assertRequestFrom: sinon.SinonStub; }; // Adjusted mock type

    beforeEach(() => {
        // Stub the helper functions
        readOneStub = sinon.stub(reads, "readOneHelper");
        readManyStub = sinon.stub(reads, "readManyHelper");
        updateOneStub = sinon.stub(updates, "updateOneHelper");

        // Ensure RequestService.get() returns a controllable mock for each test
        // The createMockContext function handles the stubbing of RequestService.get()
    });

    afterEach(() => {
        // Restore all stubs
        sinon.restore();
    });

    // --- findOne Tests ---
    describe("findOne", () => {
        const input: FindByIdInput = { id: "member-1" };
        const expectedMember = { id: "member-1", name: "Test Member", __typename: "Member" };

        it("should call readOneHelper and return a member on success", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any; // Get the mocked methods
            readOneStub.resolves(expectedMember);

            const result = await member.findOne({ input }, context, mockInfo);

            expect(result).to.deep.equal(expectedMember);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasReadPublicPermissions: true })).to.be.true;
            expect(readOneStub.calledOnceWith({ info: mockInfo, input, objectType: "Member", req: context.req })).to.be.true;
        });

        it("should throw if rate limit is exceeded", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const rateLimitError = new Error("Rate limit exceeded");
            requestServiceMock.rateLimit.rejects(rateLimitError);

            await expect(member.findOne({ input }, context, mockInfo)).to.be.rejectedWith(rateLimitError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.notCalled).to.be.true;
            expect(readOneStub.notCalled).to.be.true;
        });

        it("should throw if user lacks read permissions", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const authError = new Error("Permission denied");
            // Configure assertRequestFrom to throw for this specific call
            requestServiceMock.assertRequestFrom.withArgs(context.req, { hasReadPublicPermissions: true }).throws(authError);

            await expect(member.findOne({ input }, context, mockInfo)).to.be.rejectedWith(authError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasReadPublicPermissions: true })).to.be.true;
            expect(readOneStub.notCalled).to.be.true;
        });
    });

    // --- findMany Tests ---
    describe("findMany", () => {
        const input: MemberSearchInput = {}; // Use empty object or valid input fields
        const expectedResult = {
            results: [{ id: "member-1", name: "Test Member 1", __typename: "Member" as const }], // Add 'as const' for literal type
            total: 1,
            __typename: "MemberSearchResult" as const, // Add 'as const' for literal type
        };

        it("should call readManyHelper and return members on success", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            readManyStub.resolves(expectedResult);

            const result = await member.findMany({ input }, context, mockInfo);

            expect(result).to.deep.equal(expectedResult);
            expect(requestServiceMock.rateLimit.calledOnceWith({ maxUser: 1000, req: context.req })).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasReadPublicPermissions: true })).to.be.true;
            expect(readManyStub.calledOnceWith({ info: mockInfo, input, objectType: "Member", req: context.req })).to.be.true;
        });

        it("should throw if rate limit is exceeded", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const rateLimitError = new Error("Rate limit exceeded");
            requestServiceMock.rateLimit.rejects(rateLimitError);

            await expect(member.findMany({ input }, context, mockInfo)).to.be.rejectedWith(rateLimitError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.notCalled).to.be.true;
            expect(readManyStub.notCalled).to.be.true;
        });

        it("should throw if user lacks read permissions", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const authError = new Error("Permission denied");
            // Configure assertRequestFrom to throw for this specific call
            requestServiceMock.assertRequestFrom.withArgs(context.req, { hasReadPublicPermissions: true }).throws(authError);

            await expect(member.findMany({ input }, context, mockInfo)).to.be.rejectedWith(authError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasReadPublicPermissions: true })).to.be.true;
            expect(readManyStub.notCalled).to.be.true;
        });
    });

    // --- updateOne Tests ---
    describe("updateOne", () => {
        const input: MemberUpdateInput = { id: "member-1", isAdmin: true };
        const expectedUpdatedMember = { id: "member-1", isAdmin: true, __typename: "Member" as const };

        it("should call updateOneHelper and return the updated member on success", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            // Configure assertRequestFrom to allow write access for this call
            requestServiceMock.assertRequestFrom.withArgs(context.req, { hasWritePrivatePermissions: true }).returns(undefined);
            updateOneStub.resolves(expectedUpdatedMember);

            const result = await member.updateOne({ input }, context, mockInfo);

            expect(result).to.deep.equal(expectedUpdatedMember);
            expect(requestServiceMock.rateLimit.calledOnceWith({ maxUser: 250, req: context.req })).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasWritePrivatePermissions: true })).to.be.true;
            expect(updateOneStub.calledOnceWith({ info: mockInfo, input, objectType: "Member", req: context.req })).to.be.true;
        });

        it("should throw if rate limit is exceeded", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const rateLimitError = new Error("Rate limit exceeded");
            requestServiceMock.rateLimit.rejects(rateLimitError);

            await expect(member.updateOne({ input }, context, mockInfo)).to.be.rejectedWith(rateLimitError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.notCalled).to.be.true;
            expect(updateOneStub.notCalled).to.be.true;
        });

        it("should throw if user lacks write permissions", async () => {
            const context = createMockContext();
            requestServiceMock = RequestService.get() as any;
            const authError = new Error("Permission denied for update");
            // Configure assertRequestFrom to throw for this specific call
            requestServiceMock.assertRequestFrom.withArgs(context.req, { hasWritePrivatePermissions: true }).throws(authError);

            await expect(member.updateOne({ input }, context, mockInfo)).to.be.rejectedWith(authError);
            expect(requestServiceMock.rateLimit.calledOnce).to.be.true;
            expect(requestServiceMock.assertRequestFrom.calledOnceWith(context.req, { hasWritePrivatePermissions: true })).to.be.true;
            expect(updateOneStub.notCalled).to.be.true;
        });
    });
}); 
