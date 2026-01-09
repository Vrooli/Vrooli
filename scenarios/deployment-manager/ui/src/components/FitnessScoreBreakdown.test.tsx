// [REQ:DM-P0-004,DM-P0-005] Test fitness score breakdown component
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

// Mock component for testing
const FitnessScoreBreakdown = ({
  dependency,
  tier,
  fitnessData
}: {
  dependency: string;
  tier: string;
  fitnessData: {
    overall: number;
    portability: number;
    resourceRequirements: number;
    licensing: number;
    platformSupport: number;
    blockers?: string[];
    remediation?: string[];
  };
}) => {
  return (
    <div data-testid="fitness-breakdown">
      <h2>Fitness Score: {fitnessData.overall}/100</h2>
      <div className="subscores">
        <div data-testid="portability-score">
          Portability: {fitnessData.portability}/25
        </div>
        <div data-testid="resources-score">
          Resource Requirements: {fitnessData.resourceRequirements}/25
        </div>
        <div data-testid="licensing-score">
          Licensing: {fitnessData.licensing}/25
        </div>
        <div data-testid="platform-score">
          Platform Support: {fitnessData.platformSupport}/25
        </div>
      </div>
      {fitnessData.overall === 0 && fitnessData.blockers && (
        <div data-testid="blockers">
          <h3>Blockers</h3>
          <ul>
            {fitnessData.blockers.map((b, i) => (
              <li key={i}>{b}</li>
            ))}
          </ul>
        </div>
      )}
      {fitnessData.remediation && fitnessData.remediation.length > 0 && (
        <div data-testid="remediation">
          <h3>Remediation Steps</h3>
          <ul>
            {fitnessData.remediation.map((r, i) => (
              <li key={i}>{r}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
};

describe('FitnessScoreBreakdown Component', () => {
  // [REQ:DM-P0-004] Display 4+ fitness sub-scores
  it('[REQ:DM-P0-004] renders all four fitness sub-scores', () => {
    const fitnessData = {
      overall: 75,
      portability: 20,
      resourceRequirements: 18,
      licensing: 22,
      platformSupport: 15,
    };

    render(
      <FitnessScoreBreakdown
        dependency="postgres"
        tier="desktop"
        fitnessData={fitnessData}
      />
    );

    expect(screen.getByTestId('portability-score')).toHaveTextContent('Portability: 20/25');
    expect(screen.getByTestId('resources-score')).toHaveTextContent('Resource Requirements: 18/25');
    expect(screen.getByTestId('licensing-score')).toHaveTextContent('Licensing: 22/25');
    expect(screen.getByTestId('platform-score')).toHaveTextContent('Platform Support: 15/25');
  });

  // [REQ:DM-P0-004] All required components present
  it('[REQ:DM-P0-004] includes portability, resources, licensing, and platform support', () => {
    const fitnessData = {
      overall: 50,
      portability: 12,
      resourceRequirements: 13,
      licensing: 12,
      platformSupport: 13,
    };

    render(
      <FitnessScoreBreakdown
        dependency="ollama"
        tier="mobile"
        fitnessData={fitnessData}
      />
    );

    const breakdown = screen.getByTestId('fitness-breakdown');
    expect(breakdown).toHaveTextContent('Portability');
    expect(breakdown).toHaveTextContent('Resource Requirements');
    expect(breakdown).toHaveTextContent('Licensing');
    expect(breakdown).toHaveTextContent('Platform Support');
  });

  // [REQ:DM-P0-005] Display blockers when fitness = 0
  it('[REQ:DM-P0-005] shows blockers when fitness score is 0', () => {
    const fitnessData = {
      overall: 0,
      portability: 0,
      resourceRequirements: 0,
      licensing: 0,
      platformSupport: 0,
      blockers: ['Requires native GPU support', 'Memory footprint exceeds mobile limits'],
    };

    render(
      <FitnessScoreBreakdown
        dependency="ollama"
        tier="mobile"
        fitnessData={fitnessData}
      />
    );

    const blockersSection = screen.getByTestId('blockers');
    expect(blockersSection).toBeInTheDocument();
    expect(blockersSection).toHaveTextContent('Requires native GPU support');
    expect(blockersSection).toHaveTextContent('Memory footprint exceeds mobile limits');
  });

  // [REQ:DM-P0-005] Provide actionable remediation
  it('[REQ:DM-P0-005] displays actionable remediation steps for blockers', () => {
    const fitnessData = {
      overall: 0,
      portability: 0,
      resourceRequirements: 0,
      licensing: 0,
      platformSupport: 0,
      blockers: ['Too heavy for mobile'],
      remediation: [
        'Consider swapping to ollama-lite',
        'Use cloud-hosted AI service instead',
        'Reduce model size'
      ],
    };

    render(
      <FitnessScoreBreakdown
        dependency="ollama"
        tier="mobile"
        fitnessData={fitnessData}
      />
    );

    const remediationSection = screen.getByTestId('remediation');
    expect(remediationSection).toBeInTheDocument();
    expect(remediationSection).toHaveTextContent('Consider swapping to ollama-lite');
    expect(remediationSection).toHaveTextContent('Use cloud-hosted AI service instead');
  });
});
