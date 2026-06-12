import type { SteeringConfig } from '@/types/api';

export function deriveSteeringConfig(task: {
  auto_steer_profile_id?: string;
  steering_queue?: string[][];
  steer_set?: string[];
}): SteeringConfig {
  if (task.auto_steer_profile_id) {
    return {
      strategy: 'profile',
      profileId: task.auto_steer_profile_id,
    };
  }
  if (task.steering_queue && task.steering_queue.length > 0) {
    return {
      strategy: 'queue',
      queue: task.steering_queue,
    };
  }
  if (task.steer_set && task.steer_set.length > 0) {
    return {
      strategy: 'manual',
      manualSet: task.steer_set,
    };
  }
  return {
    strategy: 'none',
  };
}

export function extractSteeringFields(config: SteeringConfig): {
  steer_set?: string[];
  auto_steer_profile_id?: string;
  steering_queue?: string[][];
} {
  switch (config.strategy) {
    case 'profile':
      return {
        auto_steer_profile_id: config.profileId,
        steer_set: undefined,
        steering_queue: undefined,
      };
    case 'queue':
      return {
        steering_queue: config.queue ?? [],
        steer_set: undefined,
        auto_steer_profile_id: undefined,
      };
    case 'manual':
      return {
        steer_set: config.manualSet ?? [],
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
    case 'none':
    default:
      return {
        steer_set: undefined,
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
  }
}
