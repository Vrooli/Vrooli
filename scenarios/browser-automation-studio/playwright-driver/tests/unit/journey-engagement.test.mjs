import assert from 'node:assert/strict';
import test from 'node:test';
import { isActiveJourneyEngagement } from '../journeys/engagement.mjs';

const scenario = 'browser-automation-studio';
const slug = 'bas-goal';

test('accepts an exact instance engagement in Mode', () => {
  assert.equal(
    isActiveJourneyEngagement(
      { Scenario: scenario, Slug: slug, Mode: 'shadow' },
      scenario,
      slug,
      'shadow'
    ),
    true
  );
});

test('accepts an exact shadow Variant only with its matching instance key', () => {
  assert.equal(
    isActiveJourneyEngagement(
      {
        Scenario: scenario,
        Slug: slug,
        Mode: 'live',
        Variant: 'shadow',
        ShadowInstanceKey: `${scenario}@shadow`,
      },
      scenario,
      slug,
      'shadow'
    ),
    true
  );
  assert.equal(
    isActiveJourneyEngagement(
      {
        Scenario: scenario,
        Slug: slug,
        Mode: 'live',
        Variant: 'shadow',
        ShadowInstanceKey: `${scenario}@other`,
      },
      scenario,
      slug,
      'shadow'
    ),
    false
  );
});

test('rejects a different scenario, engagement, or instance variant', () => {
  for (const item of [
    { Scenario: 'another-scenario', Slug: slug, Mode: 'shadow' },
    { Scenario: scenario, Slug: 'another-engagement', Mode: 'shadow' },
    {
      Scenario: scenario,
      Slug: slug,
      Mode: 'live',
      Variant: 'live',
      ShadowInstanceKey: `${scenario}@shadow`,
    },
  ]) {
    assert.equal(isActiveJourneyEngagement(item, scenario, slug, 'shadow'), false);
  }
});
