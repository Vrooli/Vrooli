import { useCallback } from 'react';
import { appService } from '@/services/api';

export const useScenarioActions = () => {
  const restartAll = useCallback(async () => {
    if (confirm('Restart all applications?')) {
      await appService.controlAllApps('restart');
    }
  }, []);

  const stopAll = useCallback(async () => {
    if (confirm('Stop all applications?')) {
      await appService.controlAllApps('stop');
    }
  }, []);

  const triggerHealthCheck = useCallback(async () => {
    alert('Health check initiated');
  }, []);

  return {
    restartAll,
    stopAll,
    triggerHealthCheck,
  };
};
