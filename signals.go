package godo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	signalsConsentBasePath       = "/v1/consent"
	signalsConsentByIDPath       = signalsConsentBasePath + "/%s"
	signalsExportsBasePath       = "/v1/signals/exports"
	signalsExportsByIDPath       = signalsExportsBasePath + "/%s"
	signalsExportsDownloadPath   = signalsExportsBasePath + "/%s/download"
	signalsExportsOptionsPath    = signalsExportsBasePath + "/options"
	signalsExportTriggerBasePath = "/v1/signals/export-trigger"
)

// SignalsService is an interface for interacting with the DigitalOcean Signals
// consent and export APIs.
//
// Consent routes live under /v1/consent (consent-gateway). They manage
// per-agent collection consent for a team.
//
// Export routes live under /v1/signals/exports (signals-api). They manage
// async bulk export jobs. Filter catalog is GET /v1/signals/exports/options.
// Download URLs are minted separately via GET /v1/signals/exports/{id}/download.
// Weekly scheduled-export opt-in is GET/PUT /v1/signals/export-trigger.
type SignalsService interface {
	ListConsents(context.Context) ([]SignalsConsent, *Response, error)
	GetConsent(context.Context, string) (*SignalsConsent, *Response, error)
	SetConsent(context.Context, string, *SignalsConsentSetRequest) (*SignalsConsent, *Response, error)

	ListExports(context.Context, *SignalsExportListOptions) ([]SignalsExport, *Response, error)
	CreateExport(context.Context, *SignalsExportCreateRequest) (*SignalsExport, *Response, error)
	GetExport(context.Context, string) (*SignalsExport, *Response, error)
	GetExportDownload(context.Context, string) (*SignalsExportDownload, *Response, error)
	GetExportOptions(context.Context) (*SignalsExportOptions, *Response, error)

	GetExportTrigger(context.Context) (*SignalsExportTrigger, *Response, error)
	UpsertExportTrigger(context.Context, *SignalsExportTriggerUpsertRequest) (*SignalsExportTrigger, *Response, error)
}

// SignalsServiceOp handles communication with the Signals-related methods of
// the DigitalOcean API.
type SignalsServiceOp struct {
	client *Client
}

var _ SignalsService = &SignalsServiceOp{}

// SignalsConsent represents a Signals collection consent record for one
// (team, agent) pair. GET of a missing row returns enabled=false with no id.
type SignalsConsent struct {
	ID        uint64    `json:"id,omitempty"`
	TeamID    uint64    `json:"team_id"`
	AgentID   string    `json:"agent_id"`
	Enabled   bool      `json:"enabled"`
	Allowed   *bool     `json:"allowed,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// SignalsConsentSetRequest is the body for PUT /v1/consent/{agent_id}.
type SignalsConsentSetRequest struct {
	Enabled bool `json:"enabled"`
}

// SignalsExport is an async bulk-export job (ExportJob).
// Timestamps are Unix epoch seconds (UTC), not RFC3339.
// Status is one of: queued, running, complete, failed, expired.
type SignalsExport struct {
	ExportID     string               `json:"export_id"`
	AgentID      *string              `json:"agent_id,omitempty"`
	Status       string               `json:"status"`
	Filters      SignalsExportFilters `json:"filters"`
	CreatedAt    int64                `json:"created_at"`
	CompletedAt  *int64               `json:"completed_at"`
	ExpiresAt    *int64               `json:"expires_at"`
	ErrorMessage *string              `json:"error_message"`
}

// SignalsExportFilters is the create-time cohort snapshot stored on the job.
// JSON uses signal_type (singular), matching POST /exports.
type SignalsExportFilters struct {
	SessionIDs     []string `json:"session_ids,omitempty"`
	SignalType     []string `json:"signal_type,omitempty"`
	SignalCategory *string  `json:"signal_category,omitempty"`
	SignalLayer    *string  `json:"signal_layer,omitempty"`
	Concerning     *bool    `json:"concerning,omitempty"`
	StartTime      *int64   `json:"start_time,omitempty"`
	EndTime        *int64   `json:"end_time,omitempty"`
}

// SignalsExportListOptions specifies optional filters for ListExports.
type SignalsExportListOptions struct {
	AgentID string `url:"agent_id,omitempty"`
	Limit   int    `url:"limit,omitempty"`
	After   string `url:"after,omitempty"`
}

// SignalsExportCreateRequest is the body for POST /v1/signals/exports.
// Unknown fields are rejected by the API (DisallowUnknownFields).
// Use signal_type, not signal_types. signal_types is only for export-trigger.
type SignalsExportCreateRequest struct {
	AgentID        string   `json:"agent_id"`
	SessionIDs     []string `json:"session_ids,omitempty"`
	SignalType     []string `json:"signal_type,omitempty"`
	SignalCategory *string  `json:"signal_category,omitempty"`
	SignalLayer    *string  `json:"signal_layer,omitempty"`
	Concerning     *bool    `json:"concerning,omitempty"`
	StartTime      *int64   `json:"start_time,omitempty"`
	EndTime        *int64   `json:"end_time,omitempty"`
}

// SignalsExportDownload is GET /v1/signals/exports/{id}/download.
type SignalsExportDownload struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   int64  `json:"expires_at"`
}

// SignalsExportOptions is GET /v1/signals/exports/options.
type SignalsExportOptions struct {
	Filters SignalsExportFilterOptions `json:"filters"`
}

// SignalsExportFilterOptions is the static detector catalog.
type SignalsExportFilterOptions struct {
	SignalType []string `json:"signal_type"`
}

// SignalsExportTrigger is GET/PUT /v1/signals/export-trigger.
// This is the only Signals export API that uses signal_types (plural).
type SignalsExportTrigger struct {
	Enabled     bool     `json:"enabled"`
	Cadence     string   `json:"cadence"`
	SignalTypes []string `json:"signal_types"`
	UpdatedAt   *int64   `json:"updated_at,omitempty"`
}

// SignalsExportTriggerUpsertRequest is PUT /v1/signals/export-trigger.
type SignalsExportTriggerUpsertRequest struct {
	Enabled     bool     `json:"enabled"`
	Cadence     string   `json:"cadence,omitempty"`
	SignalTypes []string `json:"signal_types,omitempty"`
}

type signalsConsentListResponse struct {
	TeamID   uint64           `json:"team_id"`
	Consents []SignalsConsent `json:"consents"`
}

type signalsConsentSetResponse struct {
	Consent SignalsConsent `json:"consent"`
}

type signalsExportListResponse struct {
	Edges []struct {
		Cursor string        `json:"cursor"`
		Node   SignalsExport `json:"node"`
	} `json:"edges"`
	PageInfo struct {
		HasNextPage bool    `json:"has_next_page"`
		EndCursor   *string `json:"end_cursor,omitempty"`
	} `json:"page_info"`
}

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

func (s *SignalsServiceOp) GetConsent(ctx context.Context, agentID string) (*SignalsConsent, *Response, error) {
	if agentID == "" {
		return nil, nil, errors.New("signals: agent id is required")
	}
	path := fmt.Sprintf(signalsConsentByIDPath, url.PathEscape(agentID))
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

func (s *SignalsServiceOp) SetConsent(ctx context.Context, agentID string, body *SignalsConsentSetRequest) (*SignalsConsent, *Response, error) {
	if agentID == "" {
		return nil, nil, errors.New("signals: agent id is required")
	}
	if body == nil {
		return nil, nil, errors.New("signals: set consent request is required")
	}
	path := fmt.Sprintf(signalsConsentByIDPath, url.PathEscape(agentID))
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
	out := make([]SignalsExport, 0, len(root.Edges))
	for _, e := range root.Edges {
		out = append(out, e.Node)
	}
	return out, resp, nil
}

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

func (s *SignalsServiceOp) GetExport(ctx context.Context, exportID string) (*SignalsExport, *Response, error) {
	if exportID == "" {
		return nil, nil, errors.New("signals: export id is required")
	}
	path := fmt.Sprintf(signalsExportsByIDPath, url.PathEscape(exportID))
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

func (s *SignalsServiceOp) GetExportDownload(ctx context.Context, exportID string) (*SignalsExportDownload, *Response, error) {
	if exportID == "" {
		return nil, nil, errors.New("signals: export id is required")
	}
	path := fmt.Sprintf(signalsExportsDownloadPath, url.PathEscape(exportID))
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	dl := new(SignalsExportDownload)
	resp, err := s.client.Do(ctx, req, dl)
	if err != nil {
		return nil, resp, err
	}
	return dl, resp, nil
}

func (s *SignalsServiceOp) GetExportOptions(ctx context.Context) (*SignalsExportOptions, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, signalsExportsOptionsPath, nil)
	if err != nil {
		return nil, nil, err
	}
	opts := new(SignalsExportOptions)
	resp, err := s.client.Do(ctx, req, opts)
	if err != nil {
		return nil, resp, err
	}
	return opts, resp, nil
}

func (s *SignalsServiceOp) GetExportTrigger(ctx context.Context) (*SignalsExportTrigger, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, signalsExportTriggerBasePath, nil)
	if err != nil {
		return nil, nil, err
	}
	trig := new(SignalsExportTrigger)
	resp, err := s.client.Do(ctx, req, trig)
	if err != nil {
		return nil, resp, err
	}
	return trig, resp, nil
}

func (s *SignalsServiceOp) UpsertExportTrigger(ctx context.Context, body *SignalsExportTriggerUpsertRequest) (*SignalsExportTrigger, *Response, error) {
	if body == nil {
		return nil, nil, errors.New("signals: export trigger request is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, signalsExportTriggerBasePath, body)
	if err != nil {
		return nil, nil, err
	}
	trig := new(SignalsExportTrigger)
	resp, err := s.client.Do(ctx, req, trig)
	if err != nil {
		return nil, resp, err
	}
	return trig, resp, nil
}
