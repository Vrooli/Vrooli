import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../../test-utils/renderWithProviders';
import { describe, expect, it } from 'vitest';
import {
  AwsCredentialsHelp,
  AwsS3SetupHelp,
  CloudflareR2SetupHelp,
  MinioSetupHelp,
  ProviderComparisonHelp,
  TroubleshootingHelp,
} from './index';

describe('storage provider help content', () => {
  it('renders actionable setup guidance for every supported provider', () => {
    const { rerender } = render(<AwsCredentialsHelp />);
    expect(screen.getByText('Creating IAM Credentials')).toBeInTheDocument();
    rerender(<AwsS3SetupHelp />);
    expect(screen.getByText(/S3 bucket/i)).toBeInTheDocument();
    rerender(<CloudflareR2SetupHelp />);
    expect(screen.getByText('Step 1: Find Your Account ID')).toBeInTheDocument();
    rerender(<MinioSetupHelp />);
    expect(screen.getByText('What is MinIO?')).toBeInTheDocument();
    rerender(<ProviderComparisonHelp />);
    expect(screen.getByText('Overview')).toBeInTheDocument();
    rerender(<TroubleshootingHelp />);
    expect(screen.getByText('Common Connection Issues')).toBeInTheDocument();
  });
});
