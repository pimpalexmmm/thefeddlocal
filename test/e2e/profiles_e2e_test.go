package e2e_test

import (
	"fmt"
	"io"
	"testing"
)

func TestE2E_Profiles_GetEmpty(t *testing.T) {
	base, _ := startWebServer(t)

	resp := getJSON(t, base+"/api/profiles")
	m := decodeJSON(t, resp)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if m["profiles"] != nil {
		t.Errorf("expected profiles=null on fresh server, got %v", m["profiles"])
	}
}

func TestE2E_Profiles_CreateAndGet(t *testing.T) {
	base, _ := startWebServer(t)

	body := `{"action":"create","profile":{"id":"","nickname":"Test","config":{"domain":"test.example","key":"mypass","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":5}}}`
	resp := postJSON(t, base+"/api/profiles", body)
	m := decodeJSON(t, resp)
	if resp.StatusCode != 200 {
		t.Fatalf("create profile: expected 200, got %d", resp.StatusCode)
	}
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %v", m["ok"])
	}

	resp2 := getJSON(t, base+"/api/profiles")
	m2 := decodeJSON(t, resp2)
	profs, ok := m2["profiles"].([]any)
	if !ok || len(profs) != 1 {
		t.Fatalf("expected 1 profile, got %v", m2["profiles"])
	}
	p := profs[0].(map[string]any)
	if p["nickname"] != "Test" {
		t.Errorf("nickname = %v, want Test", p["nickname"])
	}
	cfg := p["config"].(map[string]any)
	if cfg["domain"] != "test.example" {
		t.Errorf("domain = %v, want test.example", cfg["domain"])
	}
}

func TestE2E_Profiles_CreateSetsActive(t *testing.T) {
	base, _ := startWebServer(t)

	body := `{"action":"create","profile":{"id":"","nickname":"First","config":{"domain":"first.example","key":"k1","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`
	resp := postJSON(t, base+"/api/profiles", body)
	decodeJSON(t, resp)

	resp2 := getJSON(t, base+"/api/profiles")
	m2 := decodeJSON(t, resp2)
	active, _ := m2["active"].(string)
	profs := m2["profiles"].([]any)
	firstID := profs[0].(map[string]any)["id"].(string)
	if active != firstID {
		t.Errorf("first profile should be active, active=%q id=%q", active, firstID)
	}
}

func TestE2E_Profiles_UpdateNickname(t *testing.T) {
	base, _ := startWebServer(t)

	createBody := `{"action":"create","profile":{"id":"","nickname":"OldName","config":{"domain":"upd.example","key":"k1","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`
	postJSON(t, base+"/api/profiles", createBody).Body.Close()

	m := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	id := m["profiles"].([]any)[0].(map[string]any)["id"].(string)

	updateBody := fmt.Sprintf(`{"action":"update","profile":{"id":%q,"nickname":"NewName","config":{"domain":"upd.example","key":"k1","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`, id)
	resp := postJSON(t, base+"/api/profiles", updateBody)
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("update: expected 200, got %d body=%s", resp.StatusCode, body)
	}
	resp.Body.Close()

	m2 := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	nick := m2["profiles"].([]any)[0].(map[string]any)["nickname"].(string)
	if nick != "NewName" {
		t.Errorf("nickname after update = %q, want NewName", nick)
	}
}

func TestE2E_Profiles_Delete(t *testing.T) {
	base, _ := startWebServer(t)

	postJSON(t, base+"/api/profiles", `{"action":"create","profile":{"id":"","nickname":"ToDelete","config":{"domain":"del.example","key":"k","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`).Body.Close()
	m := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	id := m["profiles"].([]any)[0].(map[string]any)["id"].(string)

	delBody := fmt.Sprintf(`{"action":"delete","profile":{"id":%q}}`, id)
	resp := postJSON(t, base+"/api/profiles", delBody)
	if resp.StatusCode != 200 {
		t.Fatalf("delete: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	m2 := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	if profs := m2["profiles"]; profs != nil {
		if list, ok := profs.([]any); ok && len(list) != 0 {
			t.Errorf("expected 0 profiles after delete, got %d", len(list))
		}
	}
}

func TestE2E_Profiles_Switch(t *testing.T) {
	base, _ := startWebServer(t)

	postJSON(t, base+"/api/profiles", `{"action":"create","profile":{"id":"","nickname":"A","config":{"domain":"a.example","key":"k","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`).Body.Close()
	postJSON(t, base+"/api/profiles", `{"action":"create","profile":{"id":"","nickname":"B","config":{"domain":"b.example","key":"k","resolvers":["127.0.0.1:9999"],"queryMode":"single","rateLimit":0}}}`).Body.Close()

	m := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	profs := m["profiles"].([]any)
	if len(profs) < 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profs))
	}
	idB := profs[1].(map[string]any)["id"].(string)

	switchBody := fmt.Sprintf(`{"id":%q}`, idB)
	resp := postJSON(t, base+"/api/profiles/switch", switchBody)
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("switch: expected 200, got %d body=%s", resp.StatusCode, body)
	}
	resp.Body.Close()

	m2 := decodeJSON(t, getJSON(t, base+"/api/profiles"))
	if m2["active"] != idB {
		t.Errorf("active after switch = %v, want %q", m2["active"], idB)
	}
}

func TestE2E_Profiles_InvalidAction(t *testing.T) {
	base, _ := startWebServer(t)

	resp := postJSON(t, base+"/api/profiles", `{"action":"bogus","profile":{}}`)
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("bogus action: expected 400, got %d", resp.StatusCode)
	}
}

func TestE2E_Profiles_SwitchNotFound(t *testing.T) {
	base, _ := startWebServer(t)

	resp := postJSON(t, base+"/api/profiles/switch", `{"id":"nonexistent-id"}`)
	defer resp.Body.Close()
	if resp.StatusCode != 400 && resp.StatusCode != 404 {
		t.Errorf("switch nonexistent: expected 400/404, got %d", resp.StatusCode)
	}
}
