import { useContext } from 'react';
import { UserAuthContext } from './UserAuthContext';

export function useUserAuth() {
  const context = useContext(UserAuthContext);
  if (!context) {
    throw new Error('useUserAuth must be used within UserAuthProvider');
  }
  return context;
}
