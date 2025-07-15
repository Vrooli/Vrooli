import { vi } from "vitest";

export const mockSocketService = {
  connect: vi.fn(),
  disconnect: vi.fn(),
  emit: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
  isConnected: vi.fn(() => false),
};

export const socketServiceMock = {
  SocketService: {
    get: () => mockSocketService,
  },
};
