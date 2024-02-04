class PrismaClientMock {
    static instance: PrismaClientMock | null = null;
    static customMocks: any = {};

    constructor() {
        if (!PrismaClientMock.instance) {
            PrismaClientMock.instance = this;
            this.$connect = jest.fn().mockResolvedValue(undefined);
            this.$disconnect = jest.fn().mockResolvedValue(undefined);
            // Mock other PrismaClient methods as necessary
        }

        // Initialize all models with basic mocks or custom mocks if provided
        Object.assign(this, PrismaClientMock.customMocks);

        return PrismaClientMock.instance;
    }

    static injectMocks(customMocks: any) {
        PrismaClientMock.customMocks = customMocks;
        PrismaClientMock.instance = new PrismaClientMock(); // Reinitialize the instance with new mocks
    }

    static resetMocks() {
        PrismaClientMock.customMocks = {};
        PrismaClientMock.instance = null; // Allow creating a new instance in the next test
    }

    // Define $connect, $disconnect, and other methods as class properties
    $connect: jest.Mock;
    $disconnect: jest.Mock;
}

export default {
    PrismaClient: PrismaClientMock,
};
