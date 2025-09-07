#!/bin/bash

# Referral Program Generator - Pattern Generation Script
# Generates standardized referral implementation patterns for scenarios

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS] SCENARIO_ANALYSIS_JSON

Generate standardized referral implementation pattern from scenario analysis.

OPTIONS:
    -r, --commission-rate RATE  Commission rate (decimal, e.g., 0.1 for 10%)
    -o, --output-dir DIR        Output directory for generated files
    -p, --preview              Preview mode - don't write files, just show plan
    -v, --verbose              Enable verbose output
    -h, --help                 Show this help message

EXAMPLES:
    $0 analysis.json
    $0 --commission-rate 0.15 --output-dir /tmp/referral analysis.json
    $0 --preview analysis.json

EOF
}

# Default values
COMMISSION_RATE=""
OUTPUT_DIR=""
PREVIEW_MODE=false
VERBOSE=false
ANALYSIS_FILE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--commission-rate)
            COMMISSION_RATE="$2"
            shift 2
            ;;
        -o|--output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--preview)
            PREVIEW_MODE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            ANALYSIS_FILE="$1"
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z "$ANALYSIS_FILE" ]]; then
    log_error "Analysis file is required"
    usage
    exit 1
fi

if [[ ! -f "$ANALYSIS_FILE" ]]; then
    log_error "Analysis file not found: $ANALYSIS_FILE"
    exit 1
fi

# Load analysis data
ANALYSIS_DATA=$(cat "$ANALYSIS_FILE")

# Extract key information
SCENARIO_PATH=$(echo "$ANALYSIS_DATA" | jq -r '.scenario_path')
BRAND_NAME=$(echo "$ANALYSIS_DATA" | jq -r '.branding.brand_name')
PRIMARY_COLOR=$(echo "$ANALYSIS_DATA" | jq -r '.branding.colors.primary')
SECONDARY_COLOR=$(echo "$ANALYSIS_DATA" | jq -r '.branding.colors.secondary')
HAS_API=$(echo "$ANALYSIS_DATA" | jq -r '.structure.has_api')
HAS_UI=$(echo "$ANALYSIS_DATA" | jq -r '.structure.has_ui')
HAS_DATABASE=$(echo "$ANALYSIS_DATA" | jq -r '.structure.has_database')
HAS_EXISTING_REFERRAL=$(echo "$ANALYSIS_DATA" | jq -r '.structure.has_existing_referral')
API_FRAMEWORK=$(echo "$ANALYSIS_DATA" | jq -r '.structure.api_framework')
UI_FRAMEWORK=$(echo "$ANALYSIS_DATA" | jq -r '.structure.ui_framework')

# Set defaults
if [[ -z "$OUTPUT_DIR" ]]; then
    OUTPUT_DIR="${SCENARIO_PATH}/src/referral"
fi

if [[ -z "$COMMISSION_RATE" ]]; then
    # Default commission rates by pricing model
    local pricing_model
    pricing_model=$(echo "$ANALYSIS_DATA" | jq -r '.pricing.model')
    case "$pricing_model" in
        "subscription") COMMISSION_RATE="0.30" ;;  # 30% for recurring revenue
        "one-time") COMMISSION_RATE="0.10" ;;      # 10% for one-time purchases
        *) COMMISSION_RATE="0.20" ;;               # 20% default
    esac
fi

if [[ "$VERBOSE" == true ]]; then
    log_info "Configuration:"
    log_info "  Scenario: $SCENARIO_PATH"
    log_info "  Brand: $BRAND_NAME"
    log_info "  Commission Rate: $(echo "$COMMISSION_RATE * 100" | bc)%"
    log_info "  Output Directory: $OUTPUT_DIR"
    log_info "  Preview Mode: $PREVIEW_MODE"
fi

# Check for existing referral implementation
if [[ "$HAS_EXISTING_REFERRAL" == "true" ]]; then
    log_warning "Existing referral implementation detected. This may overwrite existing code."
    if [[ "$PREVIEW_MODE" == false ]]; then
        read -p "Continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Operation cancelled"
            exit 0
        fi
    fi
fi

# Generate configuration file
generate_config() {
    local config_content
    config_content=$(jq -n \
        --arg brand_name "$BRAND_NAME" \
        --argjson commission_rate "$COMMISSION_RATE" \
        --arg primary_color "$PRIMARY_COLOR" \
        --arg secondary_color "$SECONDARY_COLOR" \
        --arg tracking_domain "localhost" \
        '{
            version: "1.0",
            brand: {
                name: $brand_name,
                colors: {
                    primary: $primary_color,
                    secondary: $secondary_color
                }
            },
            commission: {
                rate: $commission_rate,
                currency: "USD",
                minimum_payout: 25.00,
                payout_schedule: "monthly"
            },
            tracking: {
                domain: $tracking_domain,
                cookie_duration_days: 30,
                attribution_window_days: 30
            },
            fraud_protection: {
                rate_limit_per_ip: 100,
                suspicious_conversion_threshold: 10,
                require_email_verification: true
            }
        }')
    echo "$config_content"
}

# Generate API endpoints (Go)
generate_api_go() {
    cat << 'EOF'
package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "crypto/rand"
    "encoding/hex"
    
    "github.com/gorilla/mux"
    "github.com/google/uuid"
    _ "github.com/lib/pq"
)

// Referral data structures
type ReferralProgram struct {
    ID            string    `json:"id" db:"id"`
    ScenarioName  string    `json:"scenario_name" db:"scenario_name"`
    CommissionRate float64  `json:"commission_rate" db:"commission_rate"`
    TrackingCode  string    `json:"tracking_code" db:"tracking_code"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type ReferralLink struct {
    ID          string    `json:"id" db:"id"`
    ProgramID   string    `json:"program_id" db:"program_id"`
    ReferrerID  string    `json:"referrer_id" db:"referrer_id"`
    TrackingCode string   `json:"tracking_code" db:"tracking_code"`
    Clicks      int       `json:"clicks" db:"clicks"`
    Conversions int       `json:"conversions" db:"conversions"`
    Commission  float64   `json:"total_commission" db:"total_commission"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Commission struct {
    ID            string    `json:"id" db:"id"`
    LinkID        string    `json:"link_id" db:"link_id"`
    CustomerID    string    `json:"customer_id" db:"customer_id"`
    Amount        float64   `json:"amount" db:"amount"`
    Status        string    `json:"status" db:"status"`
    TransactionDate time.Time `json:"transaction_date" db:"transaction_date"`
}

// Generate unique tracking code
func generateTrackingCode() (string, error) {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// API Handlers
func createReferralLinkHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            ProgramID  string `json:"program_id"`
            ReferrerID string `json:"referrer_id"`
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        trackingCode, err := generateTrackingCode()
        if err != nil {
            http.Error(w, "Failed to generate tracking code", http.StatusInternalServerError)
            return
        }
        
        link := ReferralLink{
            ID:           uuid.New().String(),
            ProgramID:    req.ProgramID,
            ReferrerID:   req.ReferrerID,
            TrackingCode: trackingCode,
            CreatedAt:    time.Now(),
        }
        
        query := `
            INSERT INTO referral_links (id, program_id, referrer_id, tracking_code, clicks, conversions, total_commission, created_at)
            VALUES ($1, $2, $3, $4, 0, 0, 0, $5)`
        
        _, err = db.Exec(query, link.ID, link.ProgramID, link.ReferrerID, link.TrackingCode, link.CreatedAt)
        if err != nil {
            log.Printf("Database error: %v", err)
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(link)
    }
}

func trackClickHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        trackingCode := mux.Vars(r)["tracking_code"]
        
        // Update click count
        query := `UPDATE referral_links SET clicks = clicks + 1 WHERE tracking_code = $1`
        _, err := db.Exec(query, trackingCode)
        if err != nil {
            log.Printf("Error tracking click: %v", err)
        }
        
        // Set referral cookie
        cookie := &http.Cookie{
            Name:     "referral_code",
            Value:    trackingCode,
            Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
            HttpOnly: true,
            SameSite: http.SameSiteLaxMode,
        }
        http.SetCookie(w, cookie)
        
        // Redirect to main application
        redirectURL := r.URL.Query().Get("redirect")
        if redirectURL == "" {
            redirectURL = "/"
        }
        http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
    }
}

func recordConversionHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            CustomerID    string  `json:"customer_id"`
            Amount        float64 `json:"amount"`
            TrackingCode  string  `json:"tracking_code,omitempty"`
        }
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Get tracking code from request or cookie
        trackingCode := req.TrackingCode
        if trackingCode == "" {
            if cookie, err := r.Cookie("referral_code"); err == nil {
                trackingCode = cookie.Value
            }
        }
        
        if trackingCode == "" {
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"status": "no_referral"})
            return
        }
        
        // Get referral link and program info
        var link ReferralLink
        var program ReferralProgram
        
        query := `
            SELECT rl.id, rl.program_id, rl.referrer_id, rl.tracking_code,
                   rp.commission_rate
            FROM referral_links rl
            JOIN referral_programs rp ON rl.program_id = rp.id
            WHERE rl.tracking_code = $1`
        
        err := db.QueryRow(query, trackingCode).Scan(
            &link.ID, &link.ProgramID, &link.ReferrerID, &link.TrackingCode,
            &program.CommissionRate,
        )
        if err != nil {
            log.Printf("Error finding referral link: %v", err)
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(map[string]string{"status": "invalid_referral"})
            return
        }
        
        // Calculate commission
        commissionAmount := req.Amount * program.CommissionRate
        
        // Create commission record
        commission := Commission{
            ID:              uuid.New().String(),
            LinkID:          link.ID,
            CustomerID:      req.CustomerID,
            Amount:          commissionAmount,
            Status:          "pending",
            TransactionDate: time.Now(),
        }
        
        // Start transaction
        tx, err := db.Begin()
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        defer tx.Rollback()
        
        // Insert commission
        _, err = tx.Exec(`
            INSERT INTO commissions (id, link_id, customer_id, amount, status, transaction_date)
            VALUES ($1, $2, $3, $4, $5, $6)`,
            commission.ID, commission.LinkID, commission.CustomerID,
            commission.Amount, commission.Status, commission.TransactionDate,
        )
        if err != nil {
            log.Printf("Error inserting commission: %v", err)
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        
        // Update referral link stats
        _, err = tx.Exec(`
            UPDATE referral_links 
            SET conversions = conversions + 1, total_commission = total_commission + $1
            WHERE id = $2`,
            commissionAmount, link.ID,
        )
        if err != nil {
            log.Printf("Error updating referral link: %v", err)
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        
        // Commit transaction
        if err = tx.Commit(); err != nil {
            log.Printf("Error committing transaction: %v", err)
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status":           "success",
            "commission_id":    commission.ID,
            "commission_amount": commission.Amount,
        })
    }
}

// Add these handlers to your main router
func setupReferralRoutes(router *mux.Router, db *sql.DB) {
    router.HandleFunc("/api/referral/links", createReferralLinkHandler(db)).Methods("POST")
    router.HandleFunc("/api/referral/track/{tracking_code}", trackClickHandler(db)).Methods("GET")
    router.HandleFunc("/api/referral/conversions", recordConversionHandler(db)).Methods("POST")
}
EOF
}

# Generate database schema
generate_schema() {
    cat << 'EOF'
-- Referral Program Generator Database Schema
-- Version: 1.0

-- Create referral programs table
CREATE TABLE IF NOT EXISTS referral_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL,
    commission_rate DECIMAL(5,4) NOT NULL CHECK (commission_rate >= 0 AND commission_rate <= 1),
    tracking_code VARCHAR(50) UNIQUE NOT NULL,
    landing_page_url TEXT,
    branding_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create referral links table
CREATE TABLE IF NOT EXISTS referral_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    referrer_id UUID NOT NULL, -- References users table in scenario-authenticator
    tracking_code VARCHAR(32) UNIQUE NOT NULL,
    clicks INTEGER DEFAULT 0,
    conversions INTEGER DEFAULT 0,
    total_commission DECIMAL(12,2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create commissions table
CREATE TABLE IF NOT EXISTS commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL, -- References customer/user in the purchasing scenario
    amount DECIMAL(12,2) NOT NULL CHECK (amount >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'paid', 'disputed', 'cancelled')),
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    paid_date TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_referral_links_program_id ON referral_links(program_id);
CREATE INDEX IF NOT EXISTS idx_referral_links_referrer_id ON referral_links(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referral_links_tracking_code ON referral_links(tracking_code);
CREATE INDEX IF NOT EXISTS idx_commissions_link_id ON commissions(link_id);
CREATE INDEX IF NOT EXISTS idx_commissions_customer_id ON commissions(customer_id);
CREATE INDEX IF NOT EXISTS idx_commissions_status ON commissions(status);
CREATE INDEX IF NOT EXISTS idx_commissions_transaction_date ON commissions(transaction_date);

-- Create trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_referral_programs_updated_at 
    BEFORE UPDATE ON referral_programs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_referral_links_updated_at 
    BEFORE UPDATE ON referral_links 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_commissions_updated_at 
    BEFORE UPDATE ON commissions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default referral program (will be updated by the generator)
INSERT INTO referral_programs (scenario_name, commission_rate, tracking_code, branding_config)
VALUES (
    'default-scenario',
    0.20,
    'DEFAULT001',
    '{"colors": {"primary": "#007bff", "secondary": "#6c757d"}, "brand_name": "Default"}'::jsonb
) ON CONFLICT (tracking_code) DO NOTHING;
EOF
}

# Generate React UI component
generate_ui_react() {
    cat << 'EOF'
import React, { useState, useEffect } from 'react';
import './ReferralDashboard.css';

const ReferralDashboard = () => {
    const [referralData, setReferralData] = useState({
        links: [],
        stats: { totalClicks: 0, totalConversions: 0, totalCommission: 0 }
    });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchReferralData();
    }, []);

    const fetchReferralData = async () => {
        try {
            setLoading(true);
            const response = await fetch('/api/referral/dashboard');
            if (!response.ok) throw new Error('Failed to fetch referral data');
            
            const data = await response.json();
            setReferralData(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const createReferralLink = async () => {
        try {
            const response = await fetch('/api/referral/links', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ program_id: 'default', referrer_id: 'current_user' })
            });
            
            if (!response.ok) throw new Error('Failed to create referral link');
            
            const newLink = await response.json();
            setReferralData(prev => ({
                ...prev,
                links: [...prev.links, newLink]
            }));
        } catch (err) {
            setError(err.message);
        }
    };

    const copyToClipboard = (text) => {
        navigator.clipboard.writeText(text);
        // Show success message (implementation depends on your notification system)
        alert('Link copied to clipboard!');
    };

    if (loading) return <div className="referral-loading">Loading referral data...</div>;
    if (error) return <div className="referral-error">Error: {error}</div>;

    return (
        <div className="referral-dashboard">
            <div className="referral-header">
                <h2>Referral Dashboard</h2>
                <button className="create-link-btn" onClick={createReferralLink}>
                    Create New Referral Link
                </button>
            </div>

            <div className="referral-stats">
                <div className="stat-card">
                    <h3>Total Clicks</h3>
                    <div className="stat-value">{referralData.stats.totalClicks}</div>
                </div>
                <div className="stat-card">
                    <h3>Conversions</h3>
                    <div className="stat-value">{referralData.stats.totalConversions}</div>
                </div>
                <div className="stat-card">
                    <h3>Commission Earned</h3>
                    <div className="stat-value">${referralData.stats.totalCommission.toFixed(2)}</div>
                </div>
                <div className="stat-card">
                    <h3>Conversion Rate</h3>
                    <div className="stat-value">
                        {referralData.stats.totalClicks > 0 
                            ? `${((referralData.stats.totalConversions / referralData.stats.totalClicks) * 100).toFixed(1)}%`
                            : '0%'
                        }
                    </div>
                </div>
            </div>

            <div className="referral-links">
                <h3>Your Referral Links</h3>
                {referralData.links.length === 0 ? (
                    <p>No referral links yet. Create your first one above!</p>
                ) : (
                    <div className="links-grid">
                        {referralData.links.map(link => (
                            <div key={link.id} className="link-card">
                                <div className="link-info">
                                    <div className="tracking-code">#{link.tracking_code}</div>
                                    <div className="link-url" title={`${window.location.origin}/api/referral/track/${link.tracking_code}`}>
                                        {window.location.origin}/api/referral/track/{link.tracking_code}
                                    </div>
                                </div>
                                <div className="link-stats">
                                    <span>{link.clicks} clicks</span>
                                    <span>{link.conversions} conversions</span>
                                    <span>${link.total_commission.toFixed(2)} earned</span>
                                </div>
                                <button 
                                    className="copy-btn"
                                    onClick={() => copyToClipboard(`${window.location.origin}/api/referral/track/${link.tracking_code}`)}
                                >
                                    Copy Link
                                </button>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};

export default ReferralDashboard;
EOF
}

# Generate CSS for UI
generate_ui_css() {
    cat << EOF
/* Referral Dashboard Styles */
.referral-dashboard {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.referral-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;
    padding-bottom: 20px;
    border-bottom: 2px solid #e9ecef;
}

.referral-header h2 {
    margin: 0;
    color: ${PRIMARY_COLOR};
    font-size: 28px;
    font-weight: 600;
}

.create-link-btn {
    background: ${PRIMARY_COLOR};
    color: white;
    border: none;
    padding: 12px 24px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s ease;
}

.create-link-btn:hover {
    background: ${SECONDARY_COLOR};
    transform: translateY(-1px);
}

.referral-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;
    margin-bottom: 40px;
}

.stat-card {
    background: white;
    padding: 24px;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    text-align: center;
    border: 1px solid #e9ecef;
}

.stat-card h3 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 500;
    color: #6c757d;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.stat-value {
    font-size: 32px;
    font-weight: 700;
    color: ${PRIMARY_COLOR};
    margin: 0;
}

.referral-links h3 {
    margin-bottom: 20px;
    color: #495057;
    font-size: 20px;
    font-weight: 600;
}

.links-grid {
    display: grid;
    gap: 16px;
}

.link-card {
    background: white;
    padding: 20px;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    border: 1px solid #e9ecef;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
}

.link-info {
    flex: 1;
    min-width: 0;
}

.tracking-code {
    font-size: 12px;
    color: #6c757d;
    font-weight: 600;
    text-transform: uppercase;
    margin-bottom: 8px;
}

.link-url {
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 14px;
    color: #495057;
    background: #f8f9fa;
    padding: 8px 12px;
    border-radius: 6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    border: 1px solid #e9ecef;
}

.link-stats {
    display: flex;
    gap: 16px;
    align-items: center;
    flex-shrink: 0;
}

.link-stats span {
    font-size: 14px;
    color: #6c757d;
    font-weight: 500;
    white-space: nowrap;
}

.copy-btn {
    background: #28a745;
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    flex-shrink: 0;
}

.copy-btn:hover {
    background: #218838;
    transform: translateY(-1px);
}

.referral-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: #6c757d;
    font-size: 16px;
}

.referral-error {
    background: #f8d7da;
    color: #721c24;
    padding: 16px;
    border-radius: 8px;
    border: 1px solid #f5c6cb;
    margin: 20px 0;
}

/* Responsive design */
@media (max-width: 768px) {
    .referral-header {
        flex-direction: column;
        gap: 16px;
        align-items: stretch;
        text-align: center;
    }
    
    .referral-stats {
        grid-template-columns: repeat(2, 1fr);
    }
    
    .link-card {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;
    }
    
    .link-stats {
        justify-content: space-between;
    }
}

@media (max-width: 480px) {
    .referral-stats {
        grid-template-columns: 1fr;
    }
    
    .link-stats {
        flex-direction: column;
        gap: 8px;
    }
}
EOF
}

# Generate files based on scenario structure
generate_files() {
    local output_dir="$1"
    
    if [[ "$PREVIEW_MODE" == true ]]; then
        log_info "=== PREVIEW MODE - Files that would be generated ==="
        echo ""
        echo "📁 Directory Structure:"
        echo "  $output_dir/"
        echo "    ├── config.json"
        echo "    ├── api/"
        
        if [[ "$API_FRAMEWORK" == "go" ]]; then
            echo "    │   └── referral.go"
        else
            echo "    │   └── referral.js"
        fi
        
        echo "    ├── database/"
        echo "    │   └── schema.sql"
        
        if [[ "$HAS_UI" == "true" ]]; then
            echo "    └── ui/"
            if [[ "$UI_FRAMEWORK" == "react" ]]; then
                echo "        ├── ReferralDashboard.jsx"
                echo "        └── ReferralDashboard.css"
            else
                echo "        ├── referral-dashboard.html"
                echo "        └── referral-dashboard.css"
            fi
        fi
        
        echo ""
        echo "🎯 Configuration:"
        echo "  Brand Name: $BRAND_NAME"
        echo "  Commission Rate: $(echo "$COMMISSION_RATE * 100" | bc)%"
        echo "  Primary Color: $PRIMARY_COLOR"
        echo "  API Framework: $API_FRAMEWORK"
        echo "  UI Framework: $UI_FRAMEWORK"
        echo ""
        echo "Use --output-dir to specify a different location"
        echo "Remove --preview flag to generate actual files"
        return 0
    fi
    
    # Create directory structure
    mkdir -p "$output_dir"/{api,ui,database}
    
    # Generate configuration
    generate_config > "$output_dir/config.json"
    log_success "Generated config.json"
    
    # Generate API code
    if [[ "$API_FRAMEWORK" == "go" ]]; then
        generate_api_go > "$output_dir/api/referral.go"
        log_success "Generated Go API code"
    else
        log_warning "API framework '$API_FRAMEWORK' not yet supported, generated Go code anyway"
        generate_api_go > "$output_dir/api/referral.go"
    fi
    
    # Generate database schema
    generate_schema > "$output_dir/database/schema.sql"
    log_success "Generated database schema"
    
    # Generate UI components if scenario has UI
    if [[ "$HAS_UI" == "true" ]]; then
        if [[ "$UI_FRAMEWORK" == "react" ]]; then
            generate_ui_react > "$output_dir/ui/ReferralDashboard.jsx"
            generate_ui_css > "$output_dir/ui/ReferralDashboard.css"
            log_success "Generated React UI components"
        else
            generate_ui_css > "$output_dir/ui/referral-dashboard.css"
            log_success "Generated CSS (HTML component not yet implemented)"
        fi
    fi
    
    # Generate implementation guide
    cat > "$output_dir/IMPLEMENTATION_GUIDE.md" << EOF
# Referral Implementation Guide

This directory contains a complete referral program implementation for **$BRAND_NAME**.

## Configuration
- Commission Rate: $(echo "$COMMISSION_RATE * 100" | bc)%
- Brand Colors: Primary $PRIMARY_COLOR, Secondary $SECONDARY_COLOR
- Configuration file: \`config.json\`

## Database Setup
1. Run the SQL schema in \`database/schema.sql\`
2. This will create tables: referral_programs, referral_links, commissions

## API Integration
1. Add the code from \`api/referral.go\` to your existing API
2. Register the routes in your main router
3. Ensure database connection is available

## UI Integration
1. Copy UI components from \`ui/\` directory
2. Add to your existing UI framework
3. Update API endpoints if using different base URL

## Testing
1. Create a referral link: POST /api/referral/links
2. Test tracking: GET /api/referral/track/{code}
3. Record conversion: POST /api/referral/conversions

## Next Steps
1. Integrate with your existing authentication system
2. Add email notifications for new commissions
3. Create payout processing workflow
4. Add fraud detection rules

Generated on $(date) by Referral Program Generator
EOF
    
    log_success "Generated implementation guide"
    
    log_success "All files generated successfully in: $output_dir"
}

# Main execution
main() {
    if [[ "$VERBOSE" == true ]]; then
        log_info "Starting referral pattern generation"
    fi
    
    generate_files "$OUTPUT_DIR"
    
    if [[ "$PREVIEW_MODE" == false ]]; then
        log_info "Implementation files ready!"
        log_info "Next step: Review $OUTPUT_DIR/IMPLEMENTATION_GUIDE.md"
        
        if command -v tree >/dev/null 2>&1; then
            echo ""
            tree "$OUTPUT_DIR" 2>/dev/null || ls -la "$OUTPUT_DIR"
        fi
    fi
}

# Run main function
main