package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	signalsConsentBasePath = "/v1/consent"
	signalsConsentByIDPath = signalsConsentBasePath + "/%s"
	signalsExportsBasePath = "/v1/signals/exports"
	signalsExportsByIDPath = signalsExportsBasePath + "/%s"
)

// SignalsService is an interface for interacting with the DigitalOcean Signals
// consent and export APIs.
//
// Consent routes live under /v1/consent (consent-gateway). They manage
// per-agent collection consent for a team.
//
// Export routes live under /v1/signals/exports (signals-api). They manage
// bulk signal export jobs.
type SignalsService interface {
	// ListConsents returns all consent records for the authenticated team.
	ListConsents(context.Context) ([]SignalsConsent, *Response, error)
	// GetConsent returns the consent record for a single agent.
	GetConsent(context.Context, string) (*SignalsConsent, *Response, error)
	// SetConsent enables or disables signal collection for an agent.
	SetConsent(context.Context, string, *SignalsConsentSetRequest) (*SignalsConsent, *Response, error)

	// ListExports returns export jobs visible to the authenticated team.
	ListExports(context.Context, *SignalsExportListOptions) ([]SignalsExport, *Response, error)
	// CreateExport starts a bulk export job. The create is idempotent: if an
	// active export already exists for the same parameters, it is returned.
	CreateExport(context.Context, *SignalsExportCreateRequest) (*SignalsExport, *Response, error)
	// GetExport returns a single export job by ID. Poll this to wait for
	// completion; the download URL is set when the status is "completed".
	GetExport(context.Context, string) (*SignalsExport, *Response, error)
}

// SignalsServiceOp handles communication with the Signals-related methods of
// the DigitalOcean API.
type SignalsServiceOp struct {
	client *Client
}

var _ SignalsService = &SignalsServiceOp{}

// --- Consent types ---

// SignalsConsent represents a Signals collection consent record for one
// (team, agent) pair.
type SignalsConsent struct {
	ID        uint64    `json:"id"`
	TeamID    uint64    `json:"team_id"`
	AgentID   string    `json:"agent_id"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SignalsConsentSetRequest is the body for PUT /v1/consent/{agent_id}.
type SignalsConsentSetRequest struct {
	Enabled bool `json:"enabled"`
}

// --- Export types ---

// SignalsExport represents a Signals bulk export job.
type SignalsExport struct {
	ID           string    `json:"id"`
	TeamID       int64     `json:"team_id"`
	AgentID      string    `json:"agent_id"`
	Status       string    `json:"status"`
	SignalTypes  []string  `json:"signal_types"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompletedAt  *string   `json:"completed_at,omitempty"`
	DownloadURL  *string   `json:"download_url,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// SignalsExportListOptions specifies optional filters for ListExports.
type SignalsExportListOptions struct {
	AgentID string `url:"agent_id,omitempty"`
	Limit   int    `url:"limit,omitempty"`
	After   string `url:"after,omitempty"`
}

// SignalsExportCreateRequest is the body for POST /v1/signals/exports.
type SignalsExportCreateRequest struct {
	AgentID     string   `json:"agent_id"`
	SignalTypes []string `json:"signal_types,omitempty"`
	StartTime   *int64   `json:"start_time,omitempty"`
	EndTime     *int64   `json:"end_time,omitempty"`
}

// --- Internal response wrappers ---

type signalsConsentListResponse struct {
	TeamID   uint64           `json:"team_id"`
	Consents []SignalsConsent `json:"consents"`
}

type signalsConsentSetResponse struct {
	Consent SignalsConsent `json:"consent"`
}

type signalsExportListResponse struct {
	Exports  []SignalsExport `json:"exports"`
	PageInfo *struct {
		HasNextPage bool    `json:"has_next_page"`
		EndCursor   *string `json:"end_cursor,omitempty"`
	} `json:"page_info,omitempty"`
}

// --- Consent implementation ---

// ListConsents returns all consent records for the authenticated team.
func (s *SignalsServiceOp) ListConsents(ctx context.Context) ([]SignalsConsent, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, signalsConsentBasePath, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(signalsConsentListResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Consents, resp, nil
}

// GetConsent returns the consent record for a single agent.
func (s *SignalsServiceOp) GetConsent(ctx context.Context, agentID string) (*SignalsConsent, *Response, error) {
	if agentID == "" {
		return nil, nil, errors.New("signals: agent id is required")
	}
	path := fmt.Sprintf(signalsConsentByIDPath, agentID)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	consent := new(SignalsConsent)
	resp, err := s.client.Do(ctx, req, consent)
	if err != nil {
		return nil, resp, err
	}
	return consent, resp, nil
}

// SetConsent enables or disables signal collection for an agent.
func (s *SignalsServiceOp) SetConsent(ctx context.Context, agentID string, body *SignalsConsentSetRequest) (*SignalsConsent, *Response, error) {
	if agentID == "" {
		return nil, nil, errors.New("signals: agent id is required")
	}
	if body == nil {
		return nil, nil, errors.New("signals: set consent request is required")
	}
	path := fmt.Sprintf(signalsConsentByIDPath, agentID)
	req, err := s.client.NewRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(signalsConsentSetResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return &root.Consent, resp, nil
}

// --- Export implementation ---

// ListExports returns export jobs visible to the authenticated team.
func (s *SignalsServiceOp) ListExports(ctx context.Context, opt *SignalsExportListOptions) ([]SignalsExport, *Response, error) {
	path, err := addOptions(signalsExportsBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(signalsExportListResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Exports, resp, nil
}

// CreateExport starts a bulk signal export job.
func (s *SignalsServiceOp) CreateExport(ctx context.Context, body *SignalsExportCreateRequest) (*SignalsExport, *Response, error) {
	if body == nil {
		return nil, nil, errors.New("signals: create export request is required")
	}
	if body.AgentID == "" {
		return nil, nil, errors.New("signals: agent_id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, signalsExportsBasePath, body)
	if err != nil {
		return nil, nil, err
	}
	export := new(SignalsExport)
	resp, err := s.client.Do(ctx, req, export)
	if err != nil {
		return nil, resp, err
	}
	return export, resp, nil
}

// GetExport returns a single export job by ID.
func (s *SignalsServiceOp) GetExport(ctx context.Context, exportID string) (*SignalsExport, *Response, error) {
	if exportID == "" {
		return nil, nil, errors.New("signals: export id is required")
	}
	path := fmt.Sprintf(signalsExportsByIDPath, exportID)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	export := new(SignalsExport)
	resp, err := s.client.Do(ctx, req, export)
	if err != nil {
		return nil, resp, err
	}
	return export, resp, nil
}
