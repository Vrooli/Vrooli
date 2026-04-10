import { useCallback, useReducer } from 'react';
import {
  buildDefaultStorageForm,
  buildDefaultCredentialsForm,
  buildStorageUpdatePayload,
  getProviderDefaults,
  generateR2Endpoint,
  type StorageFormValues,
  type CredentialsFormValues,
  type StorageProviderId,
} from '../services/downloads.service';
import {
  getDownloadStorageAdmin,
  updateDownloadStorageAdmin,
  testDownloadStorageAdmin,
  type DownloadStorageSettingsSnapshot,
} from '../../../shared/api';

export type WizardStep = 'provider' | 'configure' | 'credentials' | 'verify';
export type TestStatus = 'idle' | 'testing' | 'success' | 'error';

interface WizardState {
  step: number;
  provider: StorageProviderId | null;
  cloudflareAccountId: string;
  form: StorageFormValues;
  credentials: CredentialsFormValues;
  existingSettings: DownloadStorageSettingsSnapshot | null;
  testStatus: TestStatus;
  testError: string | null;
  saveStatus: 'idle' | 'saving' | 'success' | 'error';
  saveError: string | null;
  loading: boolean;
  loadError: string | null;
}

type WizardAction =
  | { type: 'SET_STEP'; step: number }
  | { type: 'SET_PROVIDER'; provider: StorageProviderId }
  | { type: 'SET_CLOUDFLARE_ACCOUNT_ID'; accountId: string }
  | { type: 'SET_FORM'; form: Partial<StorageFormValues> }
  | { type: 'SET_CREDENTIALS'; credentials: Partial<CredentialsFormValues> }
  | { type: 'SET_EXISTING_SETTINGS'; settings: DownloadStorageSettingsSnapshot }
  | { type: 'SET_TEST_STATUS'; status: TestStatus; error?: string | null }
  | { type: 'SET_SAVE_STATUS'; status: 'idle' | 'saving' | 'success' | 'error'; error?: string | null }
  | { type: 'SET_LOADING'; loading: boolean; error?: string | null }
  | { type: 'RESET' };

const STEPS: WizardStep[] = ['provider', 'configure', 'credentials', 'verify'];

function createInitialState(): WizardState {
  return {
    step: 0,
    provider: null,
    cloudflareAccountId: '',
    form: buildDefaultStorageForm(),
    credentials: buildDefaultCredentialsForm(),
    existingSettings: null,
    testStatus: 'idle',
    testError: null,
    saveStatus: 'idle',
    saveError: null,
    loading: false,
    loadError: null,
  };
}

function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case 'SET_STEP':
      return { ...state, step: action.step, testStatus: 'idle', testError: null };
    case 'SET_PROVIDER': {
      const defaults = getProviderDefaults(action.provider);
      return {
        ...state,
        provider: action.provider,
        form: {
          ...state.form,
          region: defaults.region,
          endpoint: defaults.endpoint,
          forcePathStyle: defaults.forcePathStyle,
        },
        cloudflareAccountId: '',
      };
    }
    case 'SET_CLOUDFLARE_ACCOUNT_ID': {
      const endpoint = generateR2Endpoint(action.accountId);
      return {
        ...state,
        cloudflareAccountId: action.accountId,
        form: { ...state.form, endpoint },
      };
    }
    case 'SET_FORM':
      return { ...state, form: { ...state.form, ...action.form } };
    case 'SET_CREDENTIALS':
      return { ...state, credentials: { ...state.credentials, ...action.credentials } };
    case 'SET_EXISTING_SETTINGS':
      return { ...state, existingSettings: action.settings };
    case 'SET_TEST_STATUS':
      return { ...state, testStatus: action.status, testError: action.error ?? null };
    case 'SET_SAVE_STATUS':
      return { ...state, saveStatus: action.status, saveError: action.error ?? null };
    case 'SET_LOADING':
      return { ...state, loading: action.loading, loadError: action.error ?? null };
    case 'RESET':
      return createInitialState();
    default:
      return state;
  }
}

export interface UseStorageWizardProps {
  onComplete: () => void;
}

export interface UseStorageWizardReturn {
  state: WizardState;
  currentStepId: WizardStep;
  steps: typeof STEPS;
  canGoBack: boolean;
  canGoNext: boolean;
  isLastStep: boolean;
  goToStep: (step: number) => void;
  goBack: () => void;
  goNext: () => void;
  setProvider: (provider: StorageProviderId) => void;
  setCloudflareAccountId: (accountId: string) => void;
  setForm: (form: Partial<StorageFormValues>) => void;
  setCredentials: (credentials: Partial<CredentialsFormValues>) => void;
  loadExistingSettings: () => Promise<void>;
  testConnection: () => Promise<void>;
  saveSettings: () => Promise<void>;
  reset: () => void;
}

export function useStorageWizard({ onComplete }: UseStorageWizardProps): UseStorageWizardReturn {
  const [state, dispatch] = useReducer(wizardReducer, undefined, createInitialState);

  const currentStepId: WizardStep = STEPS[state.step] ?? 'provider';
  const canGoBack = state.step > 0;
  const isLastStep = state.step === STEPS.length - 1;

  const canGoNext = (() => {
    switch (currentStepId) {
      case 'provider':
        return state.provider !== null;
      case 'configure':
        return state.form.bucket.trim().length > 0;
      case 'credentials':
        // Credentials are optional if using env vars or already set
        return true;
      case 'verify':
        return false; // Final step - use save instead
      default:
        return false;
    }
  })();

  const goToStep = useCallback((step: number) => {
    if (step >= 0 && step < STEPS.length) {
      dispatch({ type: 'SET_STEP', step });
    }
  }, []);

  const goBack = useCallback(() => {
    if (canGoBack) {
      dispatch({ type: 'SET_STEP', step: state.step - 1 });
    }
  }, [canGoBack, state.step]);

  const goNext = useCallback(() => {
    if (canGoNext && !isLastStep) {
      dispatch({ type: 'SET_STEP', step: state.step + 1 });
    }
  }, [canGoNext, isLastStep, state.step]);

  const setProvider = useCallback((provider: StorageProviderId) => {
    dispatch({ type: 'SET_PROVIDER', provider });
  }, []);

  const setCloudflareAccountId = useCallback((accountId: string) => {
    dispatch({ type: 'SET_CLOUDFLARE_ACCOUNT_ID', accountId });
  }, []);

  const setForm = useCallback((form: Partial<StorageFormValues>) => {
    dispatch({ type: 'SET_FORM', form });
  }, []);

  const setCredentials = useCallback((credentials: Partial<CredentialsFormValues>) => {
    dispatch({ type: 'SET_CREDENTIALS', credentials });
  }, []);

  const loadExistingSettings = useCallback(async () => {
    dispatch({ type: 'SET_LOADING', loading: true });
    try {
      const { settings } = await getDownloadStorageAdmin();
      dispatch({ type: 'SET_EXISTING_SETTINGS', settings });

      // Pre-populate form with existing settings
      dispatch({
        type: 'SET_FORM',
        form: {
          bucket: settings.bucket ?? '',
          region: settings.region ?? '',
          endpoint: settings.endpoint ?? '',
          forcePathStyle: settings.force_path_style ?? false,
          defaultPrefix: settings.default_prefix ?? '',
          signedUrlTtlSeconds: settings.signed_url_ttl_seconds ?? 900,
          publicBaseUrl: settings.public_base_url ?? '',
        },
      });

      // Try to detect provider from existing settings
      if (settings.endpoint?.includes('.r2.cloudflarestorage.com')) {
        dispatch({ type: 'SET_PROVIDER', provider: 'cloudflare-r2' });
        // Extract account ID from endpoint
        const match = settings.endpoint.match(/https?:\/\/([^.]+)\.r2\.cloudflarestorage\.com/);
        if (match?.[1]) {
          dispatch({ type: 'SET_CLOUDFLARE_ACCOUNT_ID', accountId: match[1] });
        }
      } else if (settings.force_path_style && settings.endpoint) {
        dispatch({ type: 'SET_PROVIDER', provider: 'minio' });
      } else if (!settings.endpoint && settings.region) {
        dispatch({ type: 'SET_PROVIDER', provider: 'aws-s3' });
      } else if (settings.bucket) {
        dispatch({ type: 'SET_PROVIDER', provider: 'custom' });
      }

      dispatch({ type: 'SET_LOADING', loading: false });
    } catch (err) {
      dispatch({
        type: 'SET_LOADING',
        loading: false,
        error: err instanceof Error ? err.message : 'Failed to load settings',
      });
    }
  }, []);

  const testConnection = useCallback(async () => {
    dispatch({ type: 'SET_TEST_STATUS', status: 'testing' });
    try {
      await testDownloadStorageAdmin();
      dispatch({ type: 'SET_TEST_STATUS', status: 'success' });
    } catch (err) {
      dispatch({
        type: 'SET_TEST_STATUS',
        status: 'error',
        error: err instanceof Error ? err.message : 'Connection test failed',
      });
    }
  }, []);

  const saveSettings = useCallback(async () => {
    dispatch({ type: 'SET_SAVE_STATUS', status: 'saving' });
    try {
      const payload = buildStorageUpdatePayload(state.form, state.credentials);
      const { settings } = await updateDownloadStorageAdmin(payload);
      dispatch({ type: 'SET_EXISTING_SETTINGS', settings });
      dispatch({ type: 'SET_SAVE_STATUS', status: 'success' });
      onComplete();
    } catch (err) {
      dispatch({
        type: 'SET_SAVE_STATUS',
        status: 'error',
        error: err instanceof Error ? err.message : 'Failed to save settings',
      });
    }
  }, [state.form, state.credentials, onComplete]);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
  }, []);

  return {
    state,
    currentStepId,
    steps: STEPS,
    canGoBack,
    canGoNext,
    isLastStep,
    goToStep,
    goBack,
    goNext,
    setProvider,
    setCloudflareAccountId,
    setForm,
    setCredentials,
    loadExistingSettings,
    testConnection,
    saveSettings,
    reset,
  };
}
