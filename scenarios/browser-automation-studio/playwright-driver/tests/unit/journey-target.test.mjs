import assert from 'node:assert/strict';
import test from 'node:test';
import { resolveJourneyTarget } from '../journeys/target.mjs';

const endpoints = {
  BAS_JOURNEY_INSTANCE: 'shadow-copy',
  BAS_JOURNEY_API_URL: 'http://127.0.0.1:18101',
  BAS_JOURNEY_UI_URL: 'http://localhost:18102',
  BAS_JOURNEY_DRIVER_URL: 'http://[::1]:18103',
};

test('accepts an explicitly addressed isolated loopback instance without engagement lookup', () => {
  const target = resolveJourneyTarget(endpoints, () => {
    throw new Error('explicit targets must not consult shared instance ports');
  });

  assert.deepEqual(target, {
    instance: 'shadow-copy',
    api: 'http://127.0.0.1:18101',
    ui: 'http://localhost:18102',
    driver: 'http://[::1]:18103',
    source: 'explicit-loopback',
  });
});

test('requires all explicit service endpoints and distinct ports', () => {
  assert.throws(
    () => resolveJourneyTarget({ ...endpoints, BAS_JOURNEY_DRIVER_URL: '' }, () => ({})),
    /requires all of/
  );
  assert.throws(
    () =>
      resolveJourneyTarget(
        { ...endpoints, BAS_JOURNEY_DRIVER_URL: 'http://127.0.0.1:18102' },
        () => ({})
      ),
    /distinct ports/
  );
});

test('rejects non-loopback targets, credentials, and the live instance', () => {
  assert.throws(
    () =>
      resolveJourneyTarget(
        { ...endpoints, BAS_JOURNEY_API_URL: 'https://example.com' },
        () => ({})
      ),
    /loopback/
  );
  assert.throws(
    () =>
      resolveJourneyTarget(
        { ...endpoints, BAS_JOURNEY_API_URL: 'http://user:secret@127.0.0.1:18101' },
        () => ({})
      ),
    /credentials/
  );
  assert.throws(
    () => resolveJourneyTarget({ ...endpoints, BAS_JOURNEY_INSTANCE: 'live' }, () => ({})),
    /non-live/
  );
});

test('keeps the engagement-backed port lookup as the default target route', () => {
  let requestedInstance = '';
  const target = resolveJourneyTarget({ BAS_JOURNEY_INSTANCE: 'shadow' }, (instance) => {
    requestedInstance = instance;
    return { API_PORT: '18111', UI_PORT: '18112', PLAYWRIGHT_DRIVER_PORT: '18113' };
  });

  assert.equal(requestedInstance, 'shadow');
  assert.equal(target.source, 'scenario-port');
  assert.equal(target.api, 'http://127.0.0.1:18111');
});
