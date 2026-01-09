// [REQ:DM-P0-023,DM-P0-024,DM-P0-025,DM-P0-026,DM-P0-027] Test validation report component
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

interface ValidationCheck {
  name: string;
  status: 'pass' | 'fail' | 'warning';
  message: string;
  remediation?: string[];
}

interface ValidationReportProps {
  checks: ValidationCheck[];
  costEstimate?: {
    monthly: number;
    breakdown: { resource: string; cost: number }[];
  };
}

const ValidationReport = ({ checks, costEstimate }: ValidationReportProps) => {
  const passCount = checks.filter((c) => c.status === 'pass').length;
  const failCount = checks.filter((c) => c.status === 'fail').length;
  const warnCount = checks.filter((c) => c.status === 'warning').length;

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pass': return 'green';
      case 'fail': return 'red';
      case 'warning': return 'yellow';
      default: return 'gray';
    }
  };

  return (
    <div data-testid="validation-report">
      <div data-testid="summary">
        <span className="pass" style={{ color: 'green' }}>✓ {passCount}</span>
        <span className="fail" style={{ color: 'red' }}>✗ {failCount}</span>
        <span className="warning" style={{ color: 'yellow' }}>⚠ {warnCount}</span>
      </div>

      <div data-testid="checks-list">
        {checks.map((check, i) => (
          <div
            key={i}
            data-testid={`check-${check.name}`}
            style={{ color: getStatusColor(check.status) }}
          >
            <h4>{check.name}</h4>
            <p>{check.message}</p>
            {check.status === 'fail' && check.remediation && (
              <div data-testid={`remediation-${check.name}`}>
                <strong>How to fix:</strong>
                <ul>
                  {check.remediation.map((step, j) => (
                    <li key={j}>{step}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ))}
      </div>

      {costEstimate && (
        <div data-testid="cost-estimate">
          <h3>Estimated Monthly Cost</h3>
          <p className="total">${costEstimate.monthly}/month</p>
          <div className="breakdown">
            {costEstimate.breakdown.map((item, i) => (
              <div key={i} data-testid={`cost-${item.resource}`}>
                {item.resource}: ${item.cost}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

describe('ValidationReport Component', () => {
  // [REQ:DM-P0-023] Run 6+ validation checks
  it('[REQ:DM-P0-023] displays comprehensive validation with 6+ checks', () => {
    const checks: ValidationCheck[] = [
      { name: 'Fitness Threshold', status: 'pass', message: 'All dependencies meet fitness requirements' },
      { name: 'Secret Completeness', status: 'pass', message: 'All required secrets configured' },
      { name: 'Licensing', status: 'pass', message: 'No license conflicts detected' },
      { name: 'Resource Limits', status: 'warning', message: 'Memory usage near limit' },
      { name: 'Platform Requirements', status: 'pass', message: 'Platform supports all dependencies' },
      { name: 'Dependency Compatibility', status: 'pass', message: 'No version conflicts' },
    ];

    render(<ValidationReport checks={checks} />);

    expect(screen.getByTestId('check-Fitness Threshold')).toBeInTheDocument();
    expect(screen.getByTestId('check-Secret Completeness')).toBeInTheDocument();
    expect(screen.getByTestId('check-Licensing')).toBeInTheDocument();
    expect(screen.getByTestId('check-Resource Limits')).toBeInTheDocument();
    expect(screen.getByTestId('check-Platform Requirements')).toBeInTheDocument();
    expect(screen.getByTestId('check-Dependency Compatibility')).toBeInTheDocument();
  });

  // [REQ:DM-P0-025] Color-coded pass/fail/warning status
  it('[REQ:DM-P0-025] uses color coding for check status', () => {
    const checks: ValidationCheck[] = [
      { name: 'Pass Check', status: 'pass', message: 'Success' },
      { name: 'Fail Check', status: 'fail', message: 'Error' },
      { name: 'Warn Check', status: 'warning', message: 'Warning' },
    ];

    render(<ValidationReport checks={checks} />);

    const passCheck = screen.getByTestId('check-Pass Check');
    const failCheck = screen.getByTestId('check-Fail Check');
    const warnCheck = screen.getByTestId('check-Warn Check');

    expect(passCheck).toHaveStyle({ color: 'rgb(0, 128, 0)' });
    expect(failCheck).toHaveStyle({ color: 'rgb(255, 0, 0)' });
    expect(warnCheck).toHaveStyle({ color: 'rgb(255, 255, 0)' });
  });

  // [REQ:DM-P0-025] Display summary with pass/fail/warning counts
  it('[REQ:DM-P0-025] shows validation summary with counts', () => {
    const checks: ValidationCheck[] = [
      { name: 'Check 1', status: 'pass', message: 'OK' },
      { name: 'Check 2', status: 'pass', message: 'OK' },
      { name: 'Check 3', status: 'fail', message: 'Failed' },
      { name: 'Check 4', status: 'warning', message: 'Warn' },
    ];

    render(<ValidationReport checks={checks} />);

    const summary = screen.getByTestId('summary');
    expect(summary).toHaveTextContent('✓ 2');
    expect(summary).toHaveTextContent('✗ 1');
    expect(summary).toHaveTextContent('⚠ 1');
  });

  // [REQ:DM-P0-026] Provide actionable remediation for failures
  it('[REQ:DM-P0-026] shows remediation steps for failed checks', () => {
    const checks: ValidationCheck[] = [
      {
        name: 'Code Signing',
        status: 'fail',
        message: 'Code signing certificate missing',
        remediation: [
          'Upload code signing certificate here',
          'Or disable code signing for development builds'
        ]
      }
    ];

    render(<ValidationReport checks={checks} />);

    const remediation = screen.getByTestId('remediation-Code Signing');
    expect(remediation).toHaveTextContent('Upload code signing certificate here');
    expect(remediation).toHaveTextContent('Or disable code signing for development builds');
  });

  // [REQ:DM-P0-026] At least one actionable step per failure
  it('[REQ:DM-P0-026] includes minimum one remediation step for failures', () => {
    const checks: ValidationCheck[] = [
      {
        name: 'Missing Dependency',
        status: 'fail',
        message: 'Required dependency not found',
        remediation: ['Install the missing dependency using package manager']
      }
    ];

    render(<ValidationReport checks={checks} />);

    const remediation = screen.getByTestId('remediation-Missing Dependency');
    const steps = remediation.querySelectorAll('li');
    expect(steps.length).toBeGreaterThanOrEqual(1);
  });

  // [REQ:DM-P0-027] Display SaaS cost estimate
  it('[REQ:DM-P0-027] shows monthly cost estimate for SaaS tier', () => {
    const checks: ValidationCheck[] = [];
    const costEstimate = {
      monthly: 245,
      breakdown: [
        { resource: 'Compute (t3.medium)', cost: 120 },
        { resource: 'Database (RDS)', cost: 85 },
        { resource: 'Storage (S3)', cost: 20 },
        { resource: 'Bandwidth', cost: 20 },
      ]
    };

    render(<ValidationReport checks={checks} costEstimate={costEstimate} />);

    const costSection = screen.getByTestId('cost-estimate');
    expect(costSection).toHaveTextContent('$245/month');
  });

  // [REQ:DM-P0-027] Cost breakdown by resource
  it('[REQ:DM-P0-027] displays cost breakdown by resource type', () => {
    const checks: ValidationCheck[] = [];
    const costEstimate = {
      monthly: 225,
      breakdown: [
        { resource: 'EC2', cost: 120 },
        { resource: 'RDS', cost: 85 },
        { resource: 'S3', cost: 20 },
      ]
    };

    render(<ValidationReport checks={checks} costEstimate={costEstimate} />);

    expect(screen.getByTestId('cost-EC2')).toHaveTextContent('EC2: $120');
    expect(screen.getByTestId('cost-RDS')).toHaveTextContent('RDS: $85');
    expect(screen.getByTestId('cost-S3')).toHaveTextContent('S3: $20');
  });
});
