#!/bin/bash

# Referral Pattern Template Generator
# Creates standardized referral implementation for any Vrooli scenario

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

# Template functions for different components
generate_referral_config_template() {
    local scenario_name="$1"
    local brand_name="$2"
    local commission_rate="$3"
    local primary_color="$4"
    local secondary_color="$5"
    
    cat << EOF
{
  "version": "1.0",
  "scenario": "$scenario_name",
  "referral": {
    "enabled": true,
    "commission_rate": $commission_rate,
    "minimum_payout": 25.00,
    "payout_schedule": "monthly",
    "cookie_duration_days": 30,
    "attribution_window_days": 30
  },
  "branding": {
    "name": "$brand_name",
    "colors": {
      "primary": "$primary_color",
      "secondary": "$secondary_color"
    }
  },
  "tracking": {
    "analytics_enabled": true,
    "fraud_detection": true,
    "rate_limit_per_ip": 100,
    "suspicious_conversion_threshold": 10
  },
  "notifications": {
    "email_on_conversion": true,
    "webhook_url": null,
    "slack_webhook": null
  }
}
EOF
}

generate_go_api_template() {
    local scenario_name="$1"
    
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

// Referral system structures
type ReferralSystem struct {
    db *sql.DB
    config ReferralConfig
}

type ReferralConfig struct {
    CommissionRate float64 `json:"commission_rate"`
    CookieDuration int     `json:"cookie_duration_days"`
}

type ReferralLink struct {
    ID          string    `json:"id"`
    ReferrerID  string    `json:"referrer_id"`
    TrackingCode string   `json:"tracking_code"`
    Clicks      int       `json:"clicks"`
    Conversions int       `json:"conversions"`
    Commission  float64   `json:"total_commission"`
    CreatedAt   time.Time `json:"created_at"`
    IsActive    bool      `json:"is_active"`
}

type ConversionRequest struct {
    CustomerID   string  `json:"customer_id"`
    Amount       float64 `json:"amount"`
    TrackingCode string  `json:"tracking_code,omitempty"`
}

type ConversionResponse struct {
    Status           string  `json:"status"`
    CommissionID     string  `json:"commission_id,omitempty"`
    CommissionAmount float64 `json:"commission_amount,omitempty"`
    Message          string  `json:"message"`
}

// Initialize referral system
func NewReferralSystem(db *sql.DB, config ReferralConfig) *ReferralSystem {
    return &ReferralSystem{
        db: db,
        config: config,
    }
}

// Generate secure tracking code
func (rs *ReferralSystem) generateTrackingCode() (string, error) {
    bytes := make([]byte, 12)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return "REF" + hex.EncodeToString(bytes)[:8], nil
}

// Create referral link
func (rs *ReferralSystem) CreateReferralLink(referrerID string) (*ReferralLink, error) {
    trackingCode, err := rs.generateTrackingCode()
    if err != nil {
        return nil, fmt.Errorf("failed to generate tracking code: %w", err)
    }
    
    link := &ReferralLink{
        ID:           uuid.New().String(),
        ReferrerID:   referrerID,
        TrackingCode: trackingCode,
        Clicks:       0,
        Conversions:  0,
        Commission:   0.0,
        CreatedAt:    time.Now(),
        IsActive:     true,
    }
    
    query := `
        INSERT INTO referral_links 
        (id, program_id, referrer_id, tracking_code, clicks, conversions, total_commission, is_active, created_at)
        VALUES ($1, $2, $3, $4, 0, 0, 0, true, $5)`
    
    programID := "default" // Get from config or database
    _, err = rs.db.Exec(query, link.ID, programID, link.ReferrerID, link.TrackingCode, link.CreatedAt)
    if err != nil {
        return nil, fmt.Errorf("failed to save referral link: %w", err)
    }
    
    return link, nil
}

// Track click and set referral cookie
func (rs *ReferralSystem) TrackClick(trackingCode string, w http.ResponseWriter, r *http.Request) error {
    // Update click count
    _, err := rs.db.Exec(
        "UPDATE referral_links SET clicks = clicks + 1, last_click_at = NOW() WHERE tracking_code = $1 AND is_active = true",
        trackingCode,
    )
    if err != nil {
        log.Printf("Error updating click count: %v", err)
    }
    
    // Log event for analytics
    _, err = rs.db.Exec(`
        INSERT INTO referral_events (link_id, event_type, ip_address, user_agent, referrer_url, created_at)
        SELECT id, 'click', $1, $2, $3, NOW() 
        FROM referral_links 
        WHERE tracking_code = $4`,
        getClientIP(r), r.UserAgent(), r.Header.Get("Referer"), trackingCode,
    )
    if err != nil {
        log.Printf("Error logging click event: %v", err)
    }
    
    // Set referral cookie
    cookie := &http.Cookie{
        Name:     "referral_tracking",
        Value:    trackingCode,
        Expires:  time.Now().Add(time.Duration(rs.config.CookieDuration) * 24 * time.Hour),
        HttpOnly: true,
        Secure:   r.TLS != nil,
        SameSite: http.SameSiteLaxMode,
        Path:     "/",
    }
    http.SetCookie(w, cookie)
    
    return nil
}

// Record conversion and calculate commission
func (rs *ReferralSystem) RecordConversion(req ConversionRequest, r *http.Request) (*ConversionResponse, error) {
    // Get tracking code from request or cookie
    trackingCode := req.TrackingCode
    if trackingCode == "" {
        if cookie, err := r.Cookie("referral_tracking"); err == nil {
            trackingCode = cookie.Value
        }
    }
    
    if trackingCode == "" {
        return &ConversionResponse{
            Status:  "no_referral",
            Message: "No referral tracking found",
        }, nil
    }
    
    // Get referral link info
    var linkID, referrerID string
    var commissionRate float64
    
    err := rs.db.QueryRow(`
        SELECT rl.id, rl.referrer_id, rp.commission_rate
        FROM referral_links rl
        JOIN referral_programs rp ON rl.program_id = rp.id
        WHERE rl.tracking_code = $1 AND rl.is_active = true`,
        trackingCode,
    ).Scan(&linkID, &referrerID, &commissionRate)
    
    if err == sql.ErrNoRows {
        return &ConversionResponse{
            Status:  "invalid_referral",
            Message: "Referral link not found or inactive",
        }, nil
    } else if err != nil {
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    // Check for duplicate conversion (fraud prevention)
    var existingCount int
    err = rs.db.QueryRow(`
        SELECT COUNT(*) FROM commissions 
        WHERE link_id = $1 AND customer_id = $2 AND created_at > NOW() - INTERVAL '24 hours'`,
        linkID, req.CustomerID,
    ).Scan(&existingCount)
    
    if err == nil && existingCount > 0 {
        return &ConversionResponse{
            Status:  "duplicate",
            Message: "Conversion already recorded for this customer",
        }, nil
    }
    
    // Calculate commission
    commissionAmount := req.Amount * commissionRate
    
    // Start transaction for atomic updates
    tx, err := rs.db.Begin()
    if err != nil {
        return nil, fmt.Errorf("failed to start transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Create commission record
    commissionID := uuid.New().String()
    _, err = tx.Exec(`
        INSERT INTO commissions (id, link_id, customer_id, amount, status, transaction_date)
        VALUES ($1, $2, $3, $4, 'pending', NOW())`,
        commissionID, linkID, req.CustomerID, commissionAmount,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create commission: %w", err)
    }
    
    // Update referral link statistics
    _, err = tx.Exec(`
        UPDATE referral_links 
        SET conversions = conversions + 1, 
            total_commission = total_commission + $1,
            last_conversion_at = NOW()
        WHERE id = $2`,
        commissionAmount, linkID,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to update link stats: %w", err)
    }
    
    // Log conversion event
    _, err = tx.Exec(`
        INSERT INTO referral_events (link_id, event_type, customer_id, conversion_value, created_at)
        VALUES ($1, 'conversion', $2, $3, NOW())`,
        linkID, req.CustomerID, req.Amount,
    )
    if err != nil {
        log.Printf("Error logging conversion event: %v", err) // Don't fail transaction for logging
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return &ConversionResponse{
        Status:           "success",
        CommissionID:     commissionID,
        CommissionAmount: commissionAmount,
        Message:          "Conversion recorded successfully",
    }, nil
}

// HTTP Handlers
func (rs *ReferralSystem) CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ReferrerID string `json:"referrer_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    if req.ReferrerID == "" {
        http.Error(w, "referrer_id is required", http.StatusBadRequest)
        return
    }
    
    link, err := rs.CreateReferralLink(req.ReferrerID)
    if err != nil {
        log.Printf("Error creating referral link: %v", err)
        http.Error(w, "Failed to create referral link", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(link)
}

func (rs *ReferralSystem) TrackClickHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    trackingCode := vars["tracking_code"]
    
    if trackingCode == "" {
        http.Error(w, "Tracking code required", http.StatusBadRequest)
        return
    }
    
    if err := rs.TrackClick(trackingCode, w, r); err != nil {
        log.Printf("Error tracking click: %v", err)
    }
    
    // Redirect to main app or specified URL
    redirectURL := r.URL.Query().Get("redirect")
    if redirectURL == "" {
        redirectURL = "/"
    }
    
    http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (rs *ReferralSystem) ConversionHandler(w http.ResponseWriter, r *http.Request) {
    var req ConversionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    if req.CustomerID == "" || req.Amount <= 0 {
        http.Error(w, "customer_id and positive amount are required", http.StatusBadRequest)
        return
    }
    
    response, err := rs.RecordConversion(req, r)
    if err != nil {
        log.Printf("Error recording conversion: %v", err)
        http.Error(w, "Failed to record conversion", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Utility functions
func getClientIP(r *http.Request) string {
    // Check for forwarded headers
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return ip
    }
    if ip := r.Header.Get("X-Real-Ip"); ip != "" {
        return ip
    }
    return r.RemoteAddr
}

// Setup referral routes in your main router
func SetupReferralRoutes(router *mux.Router, db *sql.DB, config ReferralConfig) {
    rs := NewReferralSystem(db, config)
    
    // API routes
    apiRouter := router.PathPrefix("/api/referral").Subrouter()
    apiRouter.HandleFunc("/links", rs.CreateLinkHandler).Methods("POST")
    apiRouter.HandleFunc("/conversions", rs.ConversionHandler).Methods("POST")
    
    // Public tracking route
    router.HandleFunc("/ref/{tracking_code}", rs.TrackClickHandler).Methods("GET")
}
EOF
}

generate_react_component_template() {
    local brand_name="$1"
    local primary_color="$2"
    local secondary_color="$3"
    
    cat << EOF
import React, { useState, useEffect } from 'react';
import './ReferralProgram.css';

const ReferralProgram = ({ userId }) => {
    const [referralData, setReferralData] = useState({
        links: [],
        totalEarnings: 0,
        totalClicks: 0,
        totalConversions: 0
    });
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        if (userId) {
            fetchReferralData();
        }
    }, [userId]);

    const fetchReferralData = async () => {
        try {
            setLoading(true);
            const response = await fetch(\`/api/referral/user/\${userId}/stats\`);
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
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    referrer_id: userId
                })
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

    const copyToClipboard = async (text) => {
        try {
            await navigator.clipboard.writeText(text);
            // Show success notification
            alert('Referral link copied to clipboard!');
        } catch (err) {
            console.error('Failed to copy:', err);
            alert('Failed to copy link');
        }
    };

    const formatCurrency = (amount) => {
        return new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD'
        }).format(amount);
    };

    if (loading) return <div className="referral-loading">Loading your referral data...</div>;
    if (error) return <div className="referral-error">Error: {error}</div>;

    return (
        <div className="referral-program">
            <div className="referral-header">
                <h2>Earn with $brand_name</h2>
                <p>Share $brand_name with friends and earn commissions on every conversion!</p>
                <button className="create-link-btn" onClick={createReferralLink}>
                    Create New Referral Link
                </button>
            </div>

            <div className="referral-stats">
                <div className="stat-card earnings">
                    <div className="stat-icon">💰</div>
                    <div className="stat-content">
                        <h3>Total Earnings</h3>
                        <div className="stat-value">{formatCurrency(referralData.totalEarnings)}</div>
                    </div>
                </div>
                
                <div className="stat-card clicks">
                    <div className="stat-icon">👆</div>
                    <div className="stat-content">
                        <h3>Total Clicks</h3>
                        <div className="stat-value">{referralData.totalClicks.toLocaleString()}</div>
                    </div>
                </div>
                
                <div className="stat-card conversions">
                    <div className="stat-icon">✅</div>
                    <div className="stat-content">
                        <h3>Conversions</h3>
                        <div className="stat-value">{referralData.totalConversions.toLocaleString()}</div>
                    </div>
                </div>
                
                <div className="stat-card rate">
                    <div className="stat-icon">📊</div>
                    <div className="stat-content">
                        <h3>Conversion Rate</h3>
                        <div className="stat-value">
                            {referralData.totalClicks > 0 
                                ? \`\${((referralData.totalConversions / referralData.totalClicks) * 100).toFixed(1)}%\`
                                : '0%'
                            }
                        </div>
                    </div>
                </div>
            </div>

            <div className="referral-links">
                <h3>Your Referral Links</h3>
                {referralData.links.length === 0 ? (
                    <div className="empty-state">
                        <p>No referral links yet. Create your first one to start earning!</p>
                    </div>
                ) : (
                    <div className="links-list">
                        {referralData.links.map(link => (
                            <div key={link.id} className="link-item">
                                <div className="link-details">
                                    <div className="link-code">#{link.tracking_code}</div>
                                    <div className="link-url">
                                        {\`\${window.location.origin}/ref/\${link.tracking_code}\`}
                                    </div>
                                    <div className="link-created">
                                        Created {new Date(link.created_at).toLocaleDateString()}
                                    </div>
                                </div>
                                
                                <div className="link-performance">
                                    <div className="performance-item">
                                        <span className="label">Clicks:</span>
                                        <span className="value">{link.clicks}</span>
                                    </div>
                                    <div className="performance-item">
                                        <span className="label">Conversions:</span>
                                        <span className="value">{link.conversions}</span>
                                    </div>
                                    <div className="performance-item">
                                        <span className="label">Earned:</span>
                                        <span className="value">{formatCurrency(link.total_commission)}</span>
                                    </div>
                                </div>
                                
                                <div className="link-actions">
                                    <button 
                                        className="copy-btn"
                                        onClick={() => copyToClipboard(\`\${window.location.origin}/ref/\${link.tracking_code}\`)}
                                    >
                                        Copy Link
                                    </button>
                                    {link.is_active ? (
                                        <span className="status-active">Active</span>
                                    ) : (
                                        <span className="status-inactive">Inactive</span>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            <div className="referral-tips">
                <h3>Tips for Success</h3>
                <ul>
                    <li>Share your links on social media for maximum reach</li>
                    <li>Write honest reviews about your experience with $brand_name</li>
                    <li>Target audiences who would benefit from our services</li>
                    <li>Track your performance and optimize your sharing strategy</li>
                </ul>
            </div>
        </div>
    );
};

export default ReferralProgram;
EOF
}

generate_css_template() {
    local primary_color="$1"
    local secondary_color="$2"
    
    cat << EOF
/* Referral Program Component Styles */
.referral-program {
    max-width: 1200px;
    margin: 0 auto;
    padding: 24px;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
    line-height: 1.6;
}

/* Header Section */
.referral-header {
    text-align: center;
    margin-bottom: 40px;
    padding: 32px;
    background: linear-gradient(135deg, ${primary_color}15 0%, ${secondary_color}15 100%);
    border-radius: 16px;
    border: 1px solid ${primary_color}20;
}

.referral-header h2 {
    margin: 0 0 12px 0;
    font-size: 32px;
    font-weight: 700;
    color: ${primary_color};
}

.referral-header p {
    margin: 0 0 24px 0;
    font-size: 18px;
    color: #666;
}

.create-link-btn {
    background: ${primary_color};
    color: white;
    border: none;
    padding: 14px 28px;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 2px 8px ${primary_color}30;
}

.create-link-btn:hover {
    background: ${secondary_color};
    transform: translateY(-2px);
    box-shadow: 0 4px 16px ${primary_color}40;
}

/* Stats Section */
.referral-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    gap: 24px;
    margin-bottom: 48px;
}

.stat-card {
    background: white;
    padding: 24px;
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    border: 1px solid #e9ecef;
    display: flex;
    align-items: center;
    gap: 16px;
    transition: all 0.2s ease;
}

.stat-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.stat-icon {
    font-size: 32px;
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: ${primary_color}10;
    border-radius: 12px;
}

.stat-content h3 {
    margin: 0 0 8px 0;
    font-size: 14px;
    font-weight: 500;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.stat-value {
    font-size: 24px;
    font-weight: 700;
    color: ${primary_color};
    margin: 0;
}

.stat-card.earnings .stat-value {
    color: #28a745;
}

.stat-card.clicks .stat-value {
    color: #007bff;
}

.stat-card.conversions .stat-value {
    color: #17a2b8;
}

.stat-card.rate .stat-value {
    color: #6f42c1;
}

/* Links Section */
.referral-links {
    margin-bottom: 48px;
}

.referral-links h3 {
    margin-bottom: 24px;
    font-size: 24px;
    font-weight: 600;
    color: #333;
}

.empty-state {
    text-align: center;
    padding: 48px;
    background: #f8f9fa;
    border-radius: 12px;
    border: 2px dashed #dee2e6;
}

.empty-state p {
    margin: 0;
    color: #6c757d;
    font-size: 16px;
}

.links-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.link-item {
    background: white;
    padding: 24px;
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    border: 1px solid #e9ecef;
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: 24px;
    align-items: center;
}

.link-details {
    min-width: 0;
}

.link-code {
    font-size: 12px;
    font-weight: 600;
    color: ${primary_color};
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
    margin-bottom: 8px;
    word-break: break-all;
    border: 1px solid #e9ecef;
}

.link-created {
    font-size: 12px;
    color: #6c757d;
}

.link-performance {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 120px;
}

.performance-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;
}

.performance-item .label {
    color: #6c757d;
    font-weight: 500;
}

.performance-item .value {
    color: #333;
    font-weight: 600;
}

.link-actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
    align-items: flex-end;
}

.copy-btn {
    background: ${secondary_color};
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
}

.copy-btn:hover {
    background: ${primary_color};
    transform: translateY(-1px);
}

.status-active {
    color: #28a745;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
}

.status-inactive {
    color: #dc3545;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
}

/* Tips Section */
.referral-tips {
    background: #f8f9fa;
    padding: 24px;
    border-radius: 12px;
    border-left: 4px solid ${primary_color};
}

.referral-tips h3 {
    margin: 0 0 16px 0;
    font-size: 18px;
    font-weight: 600;
    color: #333;
}

.referral-tips ul {
    margin: 0;
    padding: 0 0 0 20px;
}

.referral-tips li {
    margin-bottom: 8px;
    color: #555;
    line-height: 1.6;
}

/* Loading and Error States */
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

/* Responsive Design */
@media (max-width: 768px) {
    .referral-program {
        padding: 16px;
    }
    
    .referral-header {
        padding: 24px 16px;
    }
    
    .referral-header h2 {
        font-size: 24px;
    }
    
    .referral-stats {
        grid-template-columns: 1fr;
    }
    
    .link-item {
        grid-template-columns: 1fr;
        gap: 16px;
    }
    
    .link-actions {
        flex-direction: row;
        justify-content: space-between;
        align-items: center;
    }
    
    .link-performance {
        flex-direction: row;
        gap: 16px;
    }
    
    .performance-item {
        flex-direction: column;
        gap: 4px;
        text-align: center;
    }
}

@media (max-width: 480px) {
    .stat-card {
        flex-direction: column;
        text-align: center;
        gap: 12px;
    }
    
    .stat-icon {
        margin: 0 auto;
    }
    
    .link-performance {
        flex-direction: column;
        gap: 8px;
    }
    
    .performance-item {
        flex-direction: row;
        justify-content: space-between;
    }
}
EOF
}

# Function to generate complete referral pattern
generate_complete_pattern() {
    local scenario_path="$1"
    local scenario_name="$2"
    local brand_name="$3"
    local commission_rate="$4"
    local primary_color="$5"
    local secondary_color="$6"
    local api_framework="$7"
    local ui_framework="$8"
    
    local referral_dir="$scenario_path/src/referral"
    
    # Create directory structure
    mkdir -p "$referral_dir"/{config,api,ui,database,docs}
    
    # Generate configuration
    generate_referral_config_template "$scenario_name" "$brand_name" "$commission_rate" "$primary_color" "$secondary_color" > "$referral_dir/config/referral.json"
    
    # Generate API code (Go for now, expand later)
    if [[ "$api_framework" == "go" || "$api_framework" == "" ]]; then
        generate_go_api_template "$scenario_name" > "$referral_dir/api/referral.go"
    fi
    
    # Generate UI components
    if [[ "$ui_framework" == "react" || "$ui_framework" == "" ]]; then
        generate_react_component_template "$brand_name" "$primary_color" "$secondary_color" > "$referral_dir/ui/ReferralProgram.jsx"
        generate_css_template "$primary_color" "$secondary_color" > "$referral_dir/ui/ReferralProgram.css"
    fi
    
    # Generate database migrations (extend existing schema)
    cat > "$referral_dir/database/001_referral_tables.sql" << 'EOF'
-- Add referral tables to existing scenario database
-- This extends the scenario's existing database schema

-- Only create tables if they don't exist (allows for scenarios that already have referral tables)
CREATE TABLE IF NOT EXISTS scenario_referral_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL UNIQUE,
    commission_rate DECIMAL(5,4) NOT NULL DEFAULT 0.20,
    is_enabled BOOLEAN DEFAULT true,
    config_json JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default config for this scenario
INSERT INTO scenario_referral_config (scenario_name, commission_rate, config_json) 
VALUES (
    current_setting('app.scenario_name', true),
    0.20,
    '{"cookie_duration_days": 30, "attribution_window_days": 30}'
) ON CONFLICT (scenario_name) DO NOTHING;

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_scenario_referral_config_name ON scenario_referral_config(scenario_name);
EOF
    
    # Generate integration guide
    cat > "$referral_dir/docs/INTEGRATION_GUIDE.md" << EOF
# Referral System Integration Guide

This referral system has been generated for **$brand_name** ($scenario_name).

## Quick Integration

### 1. Database Setup
Run the migration in \`database/001_referral_tables.sql\` against your scenario's database.

### 2. API Integration
Add the referral routes from \`api/referral.go\` to your main API router:

\`\`\`go
import "./src/referral/api"

// In your main router setup:
SetupReferralRoutes(router, db, ReferralConfig{
    CommissionRate: $commission_rate,
    CookieDuration: 30,
})
\`\`\`

### 3. UI Integration
Import and use the React component:

\`\`\`jsx
import ReferralProgram from './src/referral/ui/ReferralProgram';

// In your component:
<ReferralProgram userId={currentUser.id} />
\`\`\`

### 4. Conversion Tracking
Add conversion tracking to your payment/signup success handlers:

\`\`\`go
// After successful payment/signup:
conversionReq := ConversionRequest{
    CustomerID: userID,
    Amount: purchaseAmount,
}

response, err := referralSystem.RecordConversion(conversionReq, r)
if err != nil {
    log.Printf("Referral conversion error: %v", err)
    // Don't fail the main transaction for referral errors
}
\`\`\`

## Configuration

Edit \`config/referral.json\` to customize:
- Commission rates
- Branding colors
- Payout settings
- Fraud detection thresholds

## Testing

1. Create a referral link: POST /api/referral/links
2. Visit link: GET /ref/{tracking_code}
3. Complete signup/purchase
4. Verify commission recorded: Check database

## Support

Generated by Referral Program Generator v1.0
For support, refer to the main scenario documentation.
EOF

    log_success "Complete referral pattern generated in: $referral_dir"
    log_info "Next steps:"
    log_info "1. Review configuration in $referral_dir/config/referral.json"
    log_info "2. Follow integration guide in $referral_dir/docs/INTEGRATION_GUIDE.md"
    log_info "3. Test the implementation with sample data"
}

# Main template generation function
main() {
    if [[ $# -lt 8 ]]; then
        echo "Usage: $0 SCENARIO_PATH SCENARIO_NAME BRAND_NAME COMMISSION_RATE PRIMARY_COLOR SECONDARY_COLOR API_FRAMEWORK UI_FRAMEWORK"
        echo ""
        echo "Example:"
        echo "  $0 /path/to/my-scenario my-scenario 'My Awesome App' 0.20 '#007bff' '#6c757d' go react"
        exit 1
    fi
    
    local scenario_path="$1"
    local scenario_name="$2"
    local brand_name="$3"
    local commission_rate="$4"
    local primary_color="$5"
    local secondary_color="$6"
    local api_framework="$7"
    local ui_framework="$8"
    
    if [[ ! -d "$scenario_path" ]]; then
        log_error "Scenario directory not found: $scenario_path"
        exit 1
    fi
    
    log_info "Generating referral pattern for $brand_name"
    log_info "Commission rate: $(echo "$commission_rate * 100" | bc)%"
    log_info "Frameworks: API=$api_framework, UI=$ui_framework"
    
    generate_complete_pattern "$scenario_path" "$scenario_name" "$brand_name" "$commission_rate" "$primary_color" "$secondary_color" "$api_framework" "$ui_framework"
    
    log_success "Referral pattern template generation completed!"
}

# Only run main if script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi