class PrismaClientMock {
    static instance: PrismaClientMock | null = null;

    constructor() {
        if (!PrismaClientMock.instance) {
            PrismaClientMock.instance = this;
            this.$connect = jest.fn().mockResolvedValue(undefined);
            this.$disconnect = jest.fn().mockResolvedValue(undefined);
            // Mock other PrismaClient methods as necessary
        }

        return PrismaClientMock.instance;
    }

    // Define $connect, $disconnect, and other methods as class properties
    $connect: jest.Mock;
    $disconnect: jest.Mock;
}

export default {
    PrismaClient: PrismaClientMock,
};
