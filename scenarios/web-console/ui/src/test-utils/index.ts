export {
  apiBaseMock,
  FakeWebSocket,
  createFakeSocketPair,
  createMockTerminal,
  createTerminalStub,
  createTerminalSessionStub,
  findWriteCall,
  makeSessions,
  createMockSession,
  mockFetchSuccess,
  mockFetchError,
} from "./mocks";
export type { MockTerminal, TerminalStub } from "./mocks";
export { createTestQueryClient } from "@vrooli/api-base/testing";
export { renderWithProviders } from "./renderWithProviders";
export { setViewportWidth, setDesktopViewport, setMobileViewport } from "./viewport";
export { expectNoA11yViolations } from "@vrooli/api-base/testing";
export { asMockedClient, type MockedConnectClient } from "./mockConnectClient";

export type {
  FormalArtifact,
  FormalArtifactFreshExpectation,
  FormalArtifactTrace,
  FormalArtifactTraceStep,
  FormalArtifactTransition,
  FormalReplayAdapter,
} from "./modeltest/formal";
export {
  assertFormalArtifactFresh,
  assertFormalTracesReplay,
  assertFormalTransitionsReplay,
  transitionFromReplayAdapter,
  validateFormalArtifactFresh,
  validateFormalTracesReplay,
  validateFormalTransitionsReplay,
} from "./modeltest/formal";
