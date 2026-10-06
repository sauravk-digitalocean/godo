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

const beExportJSON = `{
	"export_id": "550e8400-e29b-41d4-a716-446655440000",
	"agent_id": "a1b2c3d4-e29b-41d4-a716-446655440000",
	"status": "complete",
	"filters": {
		"signal_type": ["MisalignmentCorrection"],
		"start_time": 1754049600,
		"end_time": 1754136000
	},
	"created_at": 1754049600,
	"completed_at": 1754049690,
	"expires_at": 1754136090,
	"error_message": null
}`

func signalsExportFixture() SignalsExport {
	agent := "a1b2c3d4-e29b-41d4-a716-446655440000"
	completed := int64(1754049690)
	expires := int64(1754136090)
	return SignalsExport{
		ExportID: "550e8400-e29b-41d4-a716-446655440000",
		AgentID:  &agent,
		Status:   "complete",
		Filters: SignalsExportFilters{
			SignalType: []string{"MisalignmentCorrection"},
			StartTime:  int64Ptr(1754049600),
			EndTime:    int64Ptr(1754136000),
		},
		CreatedAt:    1754049600,
		CompletedAt:  &completed,
		ExpiresAt:    &expires,
		ErrorMessage: nil,
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestSignalsExportCreateRequest_JSONKeyIsSignalTypeNotPlural(t *testing.T) {
	b, err := json.Marshal(SignalsExportCreateRequest{
		AgentID:    "agt",
		SignalType: []string{"MisalignmentCorrection"},
	})
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(b, &raw))
	_, hasPlural := raw["signal_types"]
	assert.False(t, hasPlural, "POST /exports must not send signal_types")
	got, ok := raw["signal_type"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"MisalignmentCorrection"}, got)
}

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

func TestSignals_GetConsent_MissingRowDefaultDeny(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/consent/missing", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"team_id":42,"agent_id":"missing","enabled":false,"allowed":false}`)
	})

	consent, resp, err := client.Signals.GetConsent(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, uint64(42), consent.TeamID)
	assert.Equal(t, "missing", consent.AgentID)
	assert.False(t, consent.Enabled)
	require.NotNil(t, consent.Allowed)
	assert.False(t, *consent.Allowed)
	assert.Zero(t, consent.ID)
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

func TestSignals_ListExports_EdgesShape(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprintf(w, `{"edges":[{"cursor":"c1","node":%s}],"page_info":{"has_next_page":false}}`, beExportJSON)
	})

	exports, resp, err := client.Signals.ListExports(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, exports, 1)
	assert.Equal(t, signalsExportFixture(), exports[0])
}

func TestSignals_ListExports_LegacyExportsKeyIsEmptyNotError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"exports":[{"export_id":"x"}]}`)
	})

	exports, _, err := client.Signals.ListExports(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, exports)
}

func TestSignals_ListExports_WithFilters(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		assert.Equal(t, "agent-abc", r.URL.Query().Get("agent_id"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "cursor-xyz", r.URL.Query().Get("after"))
		fmt.Fprint(w, `{"edges":[],"page_info":{"has_next_page":false}}`)
	})

	exports, resp, err := client.Signals.ListExports(ctx, &SignalsExportListOptions{
		AgentID: "agent-abc",
		Limit:   10,
		After:   "cursor-xyz",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, exports)
}

func TestSignals_CreateExport(t *testing.T) {
	setup()
	defer teardown()

	agent := "a1b2c3d4-e29b-41d4-a716-446655440000"
	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var raw map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		assert.Equal(t, agent, raw["agent_id"])
		assert.Equal(t, []any{"MisalignmentCorrection"}, raw["signal_type"])
		_, hasPlural := raw["signal_types"]
		assert.False(t, hasPlural)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, beExportJSON)
	})

	export, resp, err := client.Signals.CreateExport(ctx, &SignalsExportCreateRequest{
		AgentID:    agent,
		SignalType: []string{"MisalignmentCorrection"},
		StartTime:  int64Ptr(1754049600),
		EndTime:    int64Ptr(1754136000),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", export.ExportID)
	assert.Equal(t, "complete", export.Status)
	assert.Equal(t, int64(1754049600), export.CreatedAt)
}

func TestSignals_CreateExport_SessionIDs(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body SignalsExportCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []string{"sess-1"}, body.SessionIDs)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, beExportJSON)
	})

	_, resp, err := client.Signals.CreateExport(ctx, &SignalsExportCreateRequest{
		AgentID:    "a1b2c3d4-e29b-41d4-a716-446655440000",
		SessionIDs: []string{"sess-1"},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
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

	mux.HandleFunc("/v1/signals/exports/550e8400-e29b-41d4-a716-446655440000", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, beExportJSON)
	})

	export, resp, err := client.Signals.GetExport(ctx, "550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, signalsExportFixture(), *export)
	assert.Nil(t, export.ErrorMessage)
}

func TestSignals_GetExport_UnixTimestampsNotRFC3339(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports/exp-1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"export_id":"exp-1","status":"queued","filters":{},"created_at":1754049600,"completed_at":null,"expires_at":null,"error_message":null}`)
	})

	export, _, err := client.Signals.GetExport(ctx, "exp-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1754049600), export.CreatedAt)
	assert.Nil(t, export.CompletedAt)
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
		fmt.Fprint(w, `{"error":{"code":"not_found","message":"export not found","type":"not_found_error"}}`)
	})

	_, resp, err := client.Signals.GetExport(ctx, "exp-not-found")
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestSignals_GetExportDownload(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports/exp-1/download", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"download_url":"https://spaces.example/exports/a.json.gz","expires_at":1754050500}`)
	})

	dl, resp, err := client.Signals.GetExportDownload(ctx, "exp-1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "https://spaces.example/exports/a.json.gz", dl.DownloadURL)
	assert.Equal(t, int64(1754050500), dl.ExpiresAt)
}

func TestSignals_GetExportOptions(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/exports/options", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"filters":{"signal_type":["MisalignmentCorrection","StagnationRepetition"]}}`)
	})

	opts, resp, err := client.Signals.GetExportOptions(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{"MisalignmentCorrection", "StagnationRepetition"}, opts.Filters.SignalType)
}

func TestSignals_GetExportTrigger_NeverStored(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/export-trigger", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{"enabled":false,"cadence":"weekly","signal_types":[]}`)
	})

	trig, _, err := client.Signals.GetExportTrigger(ctx)
	require.NoError(t, err)
	assert.False(t, trig.Enabled)
	assert.Equal(t, "weekly", trig.Cadence)
	assert.Empty(t, trig.SignalTypes)
	assert.Nil(t, trig.UpdatedAt)
}

func TestSignals_UpsertExportTrigger(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v1/signals/export-trigger", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		var body SignalsExportTriggerUpsertRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.True(t, body.Enabled)
		assert.Equal(t, []string{"MisalignmentCorrection"}, body.SignalTypes)
		fmt.Fprint(w, `{"enabled":true,"cadence":"weekly","signal_types":["MisalignmentCorrection"],"updated_at":1754049600}`)
	})

	trig, resp, err := client.Signals.UpsertExportTrigger(ctx, &SignalsExportTriggerUpsertRequest{
		Enabled:     true,
		SignalTypes: []string{"MisalignmentCorrection"},
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, trig.Enabled)
	require.NotNil(t, trig.UpdatedAt)
	assert.Equal(t, int64(1754049600), *trig.UpdatedAt)
}

func TestSignals_UpsertExportTrigger_Nil(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Signals.UpsertExportTrigger(ctx, nil)
	require.Error(t, err)
}
