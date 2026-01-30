import { useState } from 'react';
import { Cloud, HardDrive, Server, Settings, type LucideIcon } from 'lucide-react';
import { cn } from '../../../../shared/lib/utils';
import type { StorageProviderId } from '../../services/downloads.service';
import { Callout } from '../Callout';
import { HelpModal } from './HelpModal';
import { ProviderComparisonHelp } from './help-content';

interface ProviderOption {
  id: StorageProviderId;
  name: string;
  description: string;
  icon: LucideIcon;
  features: string[];
}

const PROVIDERS: ProviderOption[] = [
  {
    id: 'aws-s3',
    name: 'AWS S3',
    description: 'Amazon Simple Storage Service',
    icon: Cloud,
    features: ['Global availability', 'Enterprise grade', 'Pay per use'],
  },
  {
    id: 'cloudflare-r2',
    name: 'Cloudflare R2',
    description: 'S3-compatible with zero egress fees',
    icon: HardDrive,
    features: ['Zero egress costs', 'Global CDN', 'S3-compatible'],
  },
  {
    id: 'minio',
    name: 'MinIO',
    description: 'Self-hosted object storage',
    icon: Server,
    features: ['Self-hosted', 'Open source', 'Full control'],
  },
  {
    id: 'custom',
    name: 'Other S3-Compatible',
    description: 'DigitalOcean Spaces, Backblaze B2, Wasabi, etc.',
    icon: Settings,
    features: ['Flexible', 'Any S3 API', 'Custom endpoint'],
  },
];

interface StepProviderProps {
  selectedProvider: StorageProviderId | null;
  onSelectProvider: (provider: StorageProviderId) => void;
}

export function StepProvider({ selectedProvider, onSelectProvider }: StepProviderProps) {
  const [showHelp, setShowHelp] = useState(false);

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-white">Choose your storage provider</h3>
        <p className="mt-1 text-sm text-slate-400">
          Select where your download artifacts will be stored
        </p>
      </div>

      <Callout
        type="info"
        message="Not sure which provider to choose? We'll help you pick the best option for your needs."
        actions={[{ label: 'Learn more', onClick: () => setShowHelp(true) }]}
      />

      <div className="grid gap-4 sm:grid-cols-2">
        {PROVIDERS.map((provider) => {
          const Icon = provider.icon;
          const isSelected = selectedProvider === provider.id;

          return (
            <button
              key={provider.id}
              type="button"
              onClick={() => onSelectProvider(provider.id)}
              className={cn(
                'relative rounded-2xl border-2 p-5 text-left transition-all duration-200',
                'hover:border-blue-500/50 hover:bg-blue-500/5',
                isSelected
                  ? 'border-blue-500 bg-blue-500/10 ring-2 ring-blue-500/20'
                  : 'border-white/10 bg-white/5'
              )}
            >
              <div className="flex items-start gap-4">
                <div
                  className={cn(
                    'flex h-12 w-12 items-center justify-center rounded-xl transition-colors',
                    isSelected ? 'bg-blue-500/20 text-blue-400' : 'bg-slate-800 text-slate-400'
                  )}
                >
                  <Icon className="h-6 w-6" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-white">{provider.name}</p>
                  <p className="mt-0.5 text-sm text-slate-400">{provider.description}</p>
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {provider.features.map((feature) => (
                      <span
                        key={feature}
                        className={cn(
                          'rounded-full px-2 py-0.5 text-xs',
                          isSelected
                            ? 'bg-blue-500/20 text-blue-300'
                            : 'bg-slate-800 text-slate-400'
                        )}
                      >
                        {feature}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              {isSelected && (
                <div className="absolute right-3 top-3">
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-blue-500">
                    <svg className="h-3 w-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path
                        fillRule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </div>
                </div>
              )}
            </button>
          );
        })}
      </div>

      <HelpModal
        open={showHelp}
        onClose={() => setShowHelp(false)}
        title="Which Storage Provider Should I Choose?"
      >
        <ProviderComparisonHelp />
      </HelpModal>
    </div>
  );
}
