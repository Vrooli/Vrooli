import { useContext } from 'react';
import { ToastContext } from './ToastContext';

/**
 * Hook to access toast notifications.
 * Must be used within a ToastProvider.
 *
 * @example
 * const { success, error } = useToast();
 *
 * const handleSave = async () => {
 *   try {
 *     await saveData();
 *     success('Changes saved successfully');
 *   } catch (err) {
 *     error('Failed to save changes');
 *   }
 * };
 */
export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}
