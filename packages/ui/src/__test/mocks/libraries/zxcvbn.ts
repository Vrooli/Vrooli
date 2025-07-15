import { vi } from "vitest";

export interface MockZxcvbnResult {
  score: number;
  feedback?: {
    warning?: string;
    suggestions?: string[];
  };
  crackTimesSeconds?: {
    offlineSlowHashing1e4PerSecond?: number;
    offlineFastHashing1e10PerSecond?: number;
    onlineThrottling100PerHour?: number;
    onlineNoThrottling10PerSecond?: number;
  };
}

export function createZxcvbnMock() {
  return vi.fn((password: string): MockZxcvbnResult => {
    if (!password) return { score: 0 };
    if (password.length < 6) return { score: 0 };
    if (password.length < 8) return { score: 1 };
    if (password.length < 10) return { score: 2 };
    if (password.length < 12) return { score: 3 };
    return { score: 4 };
  });
}

export const zxcvbnMock = {
  default: createZxcvbnMock(),
};
