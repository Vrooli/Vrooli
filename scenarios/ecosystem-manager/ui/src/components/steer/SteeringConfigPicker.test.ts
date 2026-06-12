import { describe, expect, it } from 'vitest';
import { deriveSteeringConfig, extractSteeringFields } from './SteeringConfigPicker.helpers';

describe('SteeringConfigPicker helpers', () => {
  it('derives profile strategy first', () => {
    const config = deriveSteeringConfig({
      auto_steer_profile_id: 'profile-1',
      steer_set: ['progress'],
      steering_queue: [['progress']],
    });

    expect(config).toEqual({
      strategy: 'profile',
      profileId: 'profile-1',
    });
  });

  it('derives queue strategy with set-of-sets', () => {
    const config = deriveSteeringConfig({
      steering_queue: [['progress'], ['react-coherence', 'api-steer']],
    });

    expect(config).toEqual({
      strategy: 'queue',
      queue: [['progress'], ['react-coherence', 'api-steer']],
    });
  });

  it('derives manual strategy with steer_set', () => {
    const config = deriveSteeringConfig({ steer_set: ['react-coherence', 'api-steer'] });

    expect(config).toEqual({
      strategy: 'manual',
      manualSet: ['react-coherence', 'api-steer'],
    });
  });

  it('extracts manual strategy fields', () => {
    const fields = extractSteeringFields({
      strategy: 'manual',
      manualSet: ['progress', 'test'],
    });

    expect(fields).toEqual({
      steer_set: ['progress', 'test'],
      auto_steer_profile_id: undefined,
      steering_queue: undefined,
    });
  });

  it('extracts queue strategy fields', () => {
    const fields = extractSteeringFields({
      strategy: 'queue',
      queue: [['progress'], ['test']],
    });

    expect(fields).toEqual({
      steering_queue: [['progress'], ['test']],
      steer_set: undefined,
      auto_steer_profile_id: undefined,
    });
  });
});
