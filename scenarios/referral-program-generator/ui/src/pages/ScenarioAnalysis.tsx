import React, { useState } from 'react';
import { Target, Upload, Globe, FileText, ArrowRight, CheckCircle, AlertCircle } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { referralAPI } from '../lib/api';
import toast from 'react-hot-toast';

interface AnalysisResult {
  branding: {
    colors: { primary: string; secondary: string; accent: string };
    fonts: string[];
    logo_path: string;
    brand_name: string;
  };
  pricing: {
    model: string;
    tiers: any[];
    currency: string;
  };
  structure: {
    has_api: boolean;
    has_ui: boolean;
    has_cli: boolean;
    has_database: boolean;
    has_existing_referral: boolean;
    api_framework: string;
    ui_framework: string;
  };
}

export function ScenarioAnalysis() {
  const [mode, setMode] = useState<'local' | 'deployed'>('local');
  const [scenarioPath, setScenarioPath] = useState('');
  const [deployedUrl, setDeployedUrl] = useState('');
  const [analysisResult, setAnalysisResult] = useState<AnalysisResult | null>(null);

  const analysisMutation = useMutation({
    mutationFn: (data: { scenario_path?: string; url?: string; mode: 'local' | 'deployed' }) =>
      referralAPI.analyzeScenario(data),
    onSuccess: (response) => {
      setAnalysisResult(response.data);
      toast.success('Scenario analysis completed successfully!');
    },
    onError: (error) => {
      toast.error('Analysis failed. Please check your input and try again.');
      console.error('Analysis error:', error);
    },
  });

  const handleAnalysis = () => {
    if (mode === 'local' && !scenarioPath.trim()) {
      toast.error('Please enter a scenario path');
      return;
    }
    if (mode === 'deployed' && !deployedUrl.trim()) {
      toast.error('Please enter a deployed URL');
      return;
    }

    analysisMutation.mutate({
      mode,
      scenario_path: mode === 'local' ? scenarioPath : undefined,
      url: mode === 'deployed' ? deployedUrl : undefined,
    });
  };

  return (
    <div className="scenario-analysis">
      <div className="page-header">
        <div className="page-title-section">
          <Target size={32} className="page-icon" />
          <div>
            <h1 className="page-title">Scenario Analysis</h1>
            <p className="page-description">
              Analyze Vrooli scenarios to extract branding, pricing, and structural information
            </p>
          </div>
        </div>
      </div>

      <div className="analysis-container">
        {/* Mode Selection */}
        <div className="card">
          <div className="card-header">
            <h2 className="text-xl font-semibold">Analysis Mode</h2>
            <p className="text-gray-600">Choose how to analyze your scenario</p>
          </div>
          <div className="card-body">
            <div className="mode-selector">
              <button
                className={`mode-option ${mode === 'local' ? 'active' : ''}`}
                onClick={() => setMode('local')}
              >
                <FileText size={24} />
                <div>
                  <h3>Local Scenario</h3>
                  <p>Analyze a scenario directory on your local machine</p>
                </div>
              </button>
              <button
                className={`mode-option ${mode === 'deployed' ? 'active' : ''}`}
                onClick={() => setMode('deployed')}
              >
                <Globe size={24} />
                <div>
                  <h3>Deployed Scenario</h3>
                  <p>Analyze a live deployed scenario via URL</p>
                </div>
              </button>
            </div>
          </div>
        </div>

        {/* Input Form */}
        <div className="card">
          <div className="card-header">
            <h2 className="text-xl font-semibold">
              {mode === 'local' ? 'Scenario Path' : 'Scenario URL'}
            </h2>
            <p className="text-gray-600">
              {mode === 'local'
                ? 'Enter the path to your scenario directory'
                : 'Enter the URL of your deployed scenario'
              }
            </p>
          </div>
          <div className="card-body">
            <div className="input-section">
              {mode === 'local' ? (
                <>
                  <label className="form-label">Scenario Directory Path</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="/path/to/your/scenario"
                    value={scenarioPath}
                    onChange={(e) => setScenarioPath(e.target.value)}
                  />
                  <p className="form-help">
                    Example: ../my-awesome-scenario or /home/user/scenarios/my-app
                  </p>
                </>
              ) : (
                <>
                  <label className="form-label">Deployed URL</label>
                  <input
                    type="url"
                    className="form-input"
                    placeholder="https://my-scenario.com"
                    value={deployedUrl}
                    onChange={(e) => setDeployedUrl(e.target.value)}
                  />
                  <p className="form-help">
                    Example: https://my-scenario.herokuapp.com or https://github.com/user/repo
                  </p>
                </>
              )}
            </div>
            <button
              className="btn btn-primary"
              onClick={handleAnalysis}
              disabled={analysisMutation.isPending}
            >
              {analysisMutation.isPending ? (
                <>
                  <div className="spinner" />
                  Analyzing...
                </>
              ) : (
                <>
                  <Target size={18} />
                  Analyze Scenario
                </>
              )}
            </button>
          </div>
        </div>

        {/* Analysis Results */}
        {analysisResult && (
          <div className="card">
            <div className="card-header">
              <div className="flex items-center gap-2">
                <CheckCircle size={20} className="text-success" />
                <h2 className="text-xl font-semibold">Analysis Results</h2>
              </div>
              <p className="text-gray-600">Extracted information from your scenario</p>
            </div>
            <div className="card-body">
              <div className="results-grid">
                {/* Branding Results */}
                <div className="result-section">
                  <h3 className="result-title">🎨 Branding</h3>
                  <div className="result-items">
                    <div className="result-item">
                      <span className="label">Brand Name:</span>
                      <span className="value">{analysisResult.branding.brand_name || 'Unknown'}</span>
                    </div>
                    <div className="result-item">
                      <span className="label">Primary Color:</span>
                      <div className="flex items-center gap-2">
                        <div
                          className="color-swatch"
                          style={{ backgroundColor: analysisResult.branding.colors.primary }}
                        />
                        <span className="value">{analysisResult.branding.colors.primary}</span>
                      </div>
                    </div>
                    <div className="result-item">
                      <span className="label">Fonts:</span>
                      <span className="value">
                        {analysisResult.branding.fonts.length > 0
                          ? analysisResult.branding.fonts.join(', ')
                          : 'Default fonts'}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Pricing Results */}
                <div className="result-section">
                  <h3 className="result-title">💰 Pricing</h3>
                  <div className="result-items">
                    <div className="result-item">
                      <span className="label">Model:</span>
                      <span className="value">{analysisResult.pricing.model}</span>
                    </div>
                    <div className="result-item">
                      <span className="label">Currency:</span>
                      <span className="value">{analysisResult.pricing.currency}</span>
                    </div>
                    <div className="result-item">
                      <span className="label">Tiers:</span>
                      <span className="value">{analysisResult.pricing.tiers.length} found</span>
                    </div>
                  </div>
                </div>

                {/* Structure Results */}
                <div className="result-section">
                  <h3 className="result-title">🏗️ Structure</h3>
                  <div className="result-items">
                    <div className="result-item">
                      <span className="label">API:</span>
                      <span className={`status ${analysisResult.structure.has_api ? 'success' : 'warning'}`}>
                        {analysisResult.structure.has_api ? 'Yes' : 'No'}
                        {analysisResult.structure.api_framework && ` (${analysisResult.structure.api_framework})`}
                      </span>
                    </div>
                    <div className="result-item">
                      <span className="label">UI:</span>
                      <span className={`status ${analysisResult.structure.has_ui ? 'success' : 'warning'}`}>
                        {analysisResult.structure.has_ui ? 'Yes' : 'No'}
                        {analysisResult.structure.ui_framework && ` (${analysisResult.structure.ui_framework})`}
                      </span>
                    </div>
                    <div className="result-item">
                      <span className="label">Database:</span>
                      <span className={`status ${analysisResult.structure.has_database ? 'success' : 'warning'}`}>
                        {analysisResult.structure.has_database ? 'Yes' : 'No'}
                      </span>
                    </div>
                    {analysisResult.structure.has_existing_referral && (
                      <div className="result-item warning">
                        <AlertCircle size={16} />
                        <span>Existing referral implementation detected</span>
                      </div>
                    )}
                  </div>
                </div>
              </div>

              <div className="next-steps">
                <h3 className="text-lg font-semibold mb-4">Next Steps</h3>
                <div className="flex gap-4">
                  <button className="btn btn-primary">
                    Generate Referral Program
                    <ArrowRight size={16} />
                  </button>
                  <button className="btn btn-outline">
                    Save Analysis
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      <style jsx>{`
        .scenario-analysis {
          max-width: 1000px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 2rem;
        }

        .page-title-section {
          display: flex;
          align-items: center;
          gap: 1rem;
        }

        .page-icon {
          color: var(--color-primary);
        }

        .page-title {
          font-size: 2rem;
          font-weight: 700;
          margin-bottom: 0.5rem;
          color: var(--color-gray-900);
        }

        .page-description {
          color: var(--color-gray-600);
          font-size: 1.1rem;
        }

        .analysis-container {
          display: flex;
          flex-direction: column;
          gap: 2rem;
        }

        .mode-selector {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 1rem;
        }

        .mode-option {
          display: flex;
          align-items: center;
          gap: 1rem;
          padding: 1.5rem;
          border: 2px solid var(--color-gray-200);
          border-radius: 0.75rem;
          background: var(--color-white);
          cursor: pointer;
          transition: all 0.2s ease;
          text-align: left;
        }

        .mode-option:hover {
          border-color: var(--color-primary);
          transform: translateY(-2px);
        }

        .mode-option.active {
          border-color: var(--color-primary);
          background: linear-gradient(135deg, rgba(111, 66, 193, 0.05), rgba(111, 66, 193, 0.1));
        }

        .mode-option h3 {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 0.25rem;
        }

        .mode-option p {
          color: var(--color-gray-600);
          font-size: 0.9rem;
        }

        .input-section {
          margin-bottom: 1.5rem;
        }

        .form-help {
          font-size: 0.9rem;
          color: var(--color-gray-500);
          margin-top: 0.5rem;
        }

        .results-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 1.5rem;
          margin-bottom: 2rem;
        }

        .result-section {
          background: var(--color-gray-50);
          border-radius: 0.75rem;
          padding: 1.5rem;
        }

        .result-title {
          font-size: 1.1rem;
          font-weight: 600;
          margin-bottom: 1rem;
          color: var(--color-gray-900);
        }

        .result-items {
          display: flex;
          flex-direction: column;
          gap: 0.75rem;
        }

        .result-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .result-item.warning {
          color: var(--color-warning);
          font-weight: 500;
        }

        .label {
          color: var(--color-gray-600);
          font-size: 0.9rem;
        }

        .value {
          font-weight: 500;
          color: var(--color-gray-900);
        }

        .color-swatch {
          width: 20px;
          height: 20px;
          border-radius: 4px;
          border: 1px solid var(--color-gray-300);
        }

        .status.success {
          color: var(--color-success);
        }

        .status.warning {
          color: var(--color-warning);
        }

        .next-steps {
          border-top: 1px solid var(--color-gray-200);
          padding-top: 1.5rem;
        }

        @media (max-width: 768px) {
          .mode-selector {
            grid-template-columns: 1fr;
          }
          
          .results-grid {
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
}