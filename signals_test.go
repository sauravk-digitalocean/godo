package godo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var signalsConsentFixture = SignalsConsent{
	ID:        1,
	TeamID:    42,
	AgentID:   "agent-abc",
	Enabled:   true,
	UpdatedAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
}

var signalsConsentJSON = `{
	"id": 1,
	"team_id": 42,
	"agent_id": "agent-abc",
	"enabled": true,
	"updated_at": "2026-10-01T12:00:00Z"
}`

var signalsExportFixture = SignalsExport{
	ID:          "exp-001",
	TeamID:      42,
	AgentID:     "agent-abc",
	Status:      "completed",
	SignalTypes: []string{"traces", "metrics"},
	CreatedAt:   time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 10, 1, 12, 5, 0, 0, time.UTC),
}

var signalsExportJSON = `{
	"id": "exp-001",
	"team_id": 42,
	"agent_id": "agent-abc",
	"status": "completed",
	"signal_types": ["traces", "metrics"],
	"created_at": "2026-10-01T12:00:00Z",
	"updated_at": "2026-10-01T12:05:00Z"
}`

// --- Consent tests ---

func TestSignals_ListConsents(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/consent", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprintf(w, `{"team_id":42,"consents":[%s]}`, signalsConsentJSON)
	})

	consents, resp, err := client.Signals.ListConsents(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, consents, 1)
	assert.Equal(t, signalsConsentFixture, consents[0])
}

func TestSignals_ListConsents_Empty(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/consent", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"team_id":42,"consents":[]}`)
	})

	consents, resp, err := client.Signals.ListConsents(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, consents)
}

func TestSignals_GetConsent(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/consent/agent-abc", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, signalsConsentJSON)
	})

	consent, resp, err := client.Signals.GetConsent(ctx, "agent-abc")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, signalsConsentFixture, *consent)
}

func TestSignals_GetConsent_EmptyID(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.GetConsent(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent id is required")
}

func TestSignals_SetConsent_Enable(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/consent/agent-abc", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		var body SignalsConsentSetRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.True(t, body.Enabled)
		fmt.Fprintf(w, `{"consent":%s}`, signalsConsentJSON)
	})

	consent, resp, err := client.Signals.SetConsent(ctx, "agent-abc", &SignalsConsentSetRequest{Enabled: true})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, signalsConsentFixture, *consent)
}

func TestSignals_SetConsent_Disable(t *testing.T) {
	setup()
	defer teardown()

	disabledConsent := `{
		"id": 1,
		"team_id": 42,
		"agent_id": "agent-abc",
		"enabled": false,
		"updated_at": "2026-10-01T12:00:00Z"
	}`

	mux.HandleFunc("/v1/consent/agent-abc", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		var body SignalsConsentSetRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.False(t, body.Enabled)
		fmt.Fprintf(w, `{"consent":%s}`, disabledConsent)
	})

	consent, resp, err := client.Signals.SetConsent(ctx, "agent-abc", &SignalsConsentSetRequest{Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.False(t, consent.Enabled)
}

func TestSignals_SetConsent_EmptyID(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.SetConsent(ctx, "", &SignalsConsentSetRequest{Enabled: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent id is required")
}

func TestSignals_SetConsent_NilRequest(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.SetConsent(ctx, "agent-abc", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "set consent request is required")
}

// --- Export tests ---

func TestSignals_ListExports(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprintf(w, `{"exports":[%s]}`, signalsExportJSON)
	})

	exports, resp, err := client.Signals.ListExports(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, exports, 1)
	assert.Equal(t, signalsExportFixture, exports[0])
}

func TestSignals_ListExports_WithFilters(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		assert.Equal(t, "agent-abc", r.URL.Query().Get("agent_id"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "cursor-xyz", r.URL.Query().Get("after"))
		fmt.Fprintf(w, `{"exports":[%s]}`, signalsExportJSON)
	})

	exports, resp, err := client.Signals.ListExports(ctx, &SignalsExportListOptions{
		AgentID: "agent-abc",
		Limit:   10,
		After:   "cursor-xyz",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, exports, 1)
}

func TestSignals_ListExports_Empty(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"exports":[]}`)
	})

	exports, resp, err := client.Signals.ListExports(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, exports)
}

func TestSignals_CreateExport(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body SignalsExportCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "agent-abc", body.AgentID)
		assert.Equal(t, []string{"traces", "metrics"}, body.SignalTypes)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, signalsExportJSON)
	})

	export, resp, err := client.Signals.CreateExport(ctx, &SignalsExportCreateRequest{
		AgentID:     "agent-abc",
		SignalTypes: []string{"traces", "metrics"},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, signalsExportFixture, *export)
}

func TestSignals_CreateExport_WithTimeRange(t *testing.T) {
	setup()
	defer teardown()

	start := int64(1696100000)
	end := int64(1696200000)

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body SignalsExportCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "agent-abc", body.AgentID)
		require.NotNil(t, body.StartTime)
		assert.Equal(t, start, *body.StartTime)
		require.NotNil(t, body.EndTime)
		assert.Equal(t, end, *body.EndTime)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, signalsExportJSON)
	})

	export, resp, err := client.Signals.CreateExport(ctx, &SignalsExportCreateRequest{
		AgentID:   "agent-abc",
		StartTime: &start,
		EndTime:   &end,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, signalsExportFixture, *export)
}

func TestSignals_CreateExport_NilRequest(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.CreateExport(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create export request is required")
}

func TestSignals_CreateExport_EmptyAgentID(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.CreateExport(ctx, &SignalsExportCreateRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent_id is required")
}

func TestSignals_GetExport(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports/exp-001", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, signalsExportJSON)
	})

	export, resp, err := client.Signals.GetExport(ctx, "exp-001")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, signalsExportFixture, *export)
}

func TestSignals_GetExport_EmptyID(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.GetExport(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "export id is required")
}

func TestSignals_GetExport_404(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports/exp-not-found", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"id":"not_found","message":"export not found"}`)
	})

	_, resp, err := client.Signals.GetExport(ctx, "exp-not-found")
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
