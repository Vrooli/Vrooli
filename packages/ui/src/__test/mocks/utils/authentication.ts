import { vi } from "vitest";

export interface MockAuthConfig {
  isLoggedIn?: boolean;
  userId?: string | null;
  userRoles?: string[];
  languages?: string[];
}

// Factory function for customizable auth mocks
export function createAuthMock(config: MockAuthConfig = {}) {
  const {
    isLoggedIn = false,
    userId = null,
    userRoles = [],
    languages = ["en"],
  } = config;

  return {
    getCurrentUser: vi.fn((session) => {
      // Return empty object for invalid session cases, just like the real implementation
      if (!session || !session.isLoggedIn || !Array.isArray(session.users) || session.users.length === 0) {
        return {};
      }
      
      const userData = session.users[0];
      // Check if user ID is valid - if not, return empty object
      if (!userData.id || userData.id === "invalid-id") {
        return {};
      }
      
      // Return the actual user data for valid cases
      return userData;
    }),
    checkIfLoggedIn: vi.fn((session) => {
      // If there is no session, check local storage (like the real implementation)
      if (!session) {
        return localStorage.getItem("isLoggedIn") === "true";
      }
      // Otherwise, check session
      return session.isLoggedIn;
    }),
    guestSession: {
      __typename: "Session",
      isLoggedIn: false,
      users: [],
    },
  };
}

// Default mock (guest user)
export const authMock = createAuthMock();
