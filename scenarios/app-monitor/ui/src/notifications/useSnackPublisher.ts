import { useContext } from 'react';
import { SnackPublisherContext, type SnackPublisherApi } from './snackPublisherContext';

export function useSnackPublisher(): SnackPublisherApi {
  const context = useContext(SnackPublisherContext);
  if (!context) {
    throw new Error('useSnackPublisher must be used within a SnackStackProvider');
  }
  return context;
}
