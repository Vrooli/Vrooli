// Stable policy projection for UI tests. The implementation remains owned by
// api-base so provider setup is shared across scenarios.
export { renderWithProviders, renderWithProviders as render } from '@vrooli/api-base/testing'
export {
  act,
  cleanup,
  fireEvent,
  renderHook,
  screen,
  waitFor,
  waitForElementToBeRemoved,
  within,
} from '@testing-library/react'
