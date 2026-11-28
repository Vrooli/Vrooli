// Test environment setup for Vitest + jsdom
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
import { installBrowserApiShims } from './mocks/browser';

installBrowserApiShims();

afterEach(() => {
  cleanup();
});
