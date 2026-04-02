package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"imuslab.com/zoraxy/mod/node"
)

func TestRenderNodeJoinTemplateResponseByName(t *testing.T) {
	manifest, err := loadNodeJoinTemplateManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	definition, isDynamicRequest, err := resolveNodeJoinTemplateDefinition(manifest, "standaloneJoinFlagCommand", -1)
	if err != nil {
		t.Fatalf("resolve template by name: %v", err)
	}
	if isDynamicRequest {
		t.Fatal("expected standalone join template, got dynamic request")
	}

	req := httptest.NewRequest(http.MethodGet, "https://primary.example:8443/api/nodes/join/templates", nil)
	req.Host = "primary.example:8443"
	req.Header.Set("X-Forwarded-Proto", "https")

	rendered, err := renderNodeJoinTemplateResponse(req, &node.Node{
		ID:    "node-1",
		Name:  "worker-one",
		Token: "secret-token",
	}, definition)
	if err != nil {
		t.Fatalf("render template: %v", err)
	}

	expected := "ZORAXY_NODE_NAME='worker-one' zoraxy -server 'https://primary.example:8443' -token 'secret-token'"
	if !strings.Contains(rendered.Content, expected) {
		t.Fatalf("expected rendered content to contain %q, got %q", expected, rendered.Content)
	}
}

func TestRenderNodeJoinTemplateResponseByIndex(t *testing.T) {
	manifest, err := loadNodeJoinTemplateManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	definition, isDynamicRequest, err := resolveNodeJoinTemplateDefinition(manifest, "", 1)
	if err != nil {
		t.Fatalf("resolve template by index: %v", err)
	}
	if isDynamicRequest {
		t.Fatal("expected indexed instruction, got dynamic request")
	}

	req := httptest.NewRequest(http.MethodGet, "http://primary.example:8000/api/nodes/join/templates", nil)
	req.Host = "primary.example:8000"

	rendered, err := renderNodeJoinTemplateResponse(req, &node.Node{
		ID:    "node-2",
		Host:  "worker-two",
		Token: "compose-token",
	}, definition)
	if err != nil {
		t.Fatalf("render template: %v", err)
	}

	if !strings.Contains(rendered.Content, `ZORAXY_NODE_TOKEN: "compose-token"`) {
		t.Fatalf("expected compose output to include quoted token, got %q", rendered.Content)
	}
	if !strings.Contains(rendered.Content, `FASTGEOIP: "true"`) {
		t.Fatalf("expected compose output to include FASTGEOIP, got %q", rendered.Content)
	}
}

func TestRenderNodeJoinInstructionsExcludesOptionalNodeIPOverride(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://primary.example:8443/node/api/request", nil)
	req.Host = "primary.example:8443"
	req.Header.Set("X-Forwarded-Proto", "https")

	instructions, err := renderNodeJoinInstructions(req, &node.Node{
		ID:    "node-3",
		Name:  "worker-three",
		Token: "filtered-token",
	}, map[string]struct{}{
		nodeJoinOptionalIPCommandID: {},
	})
	if err != nil {
		t.Fatalf("render filtered instruction list: %v", err)
	}

	manifest, err := loadNodeJoinTemplateManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	expectedCount := 0
	for _, definition := range manifest.Instructions {
		if definition != nil && strings.TrimSpace(definition.ID) != nodeJoinOptionalIPCommandID {
			expectedCount++
		}
	}

	if len(instructions) != expectedCount {
		t.Fatalf("expected %d filtered instructions, got %d", expectedCount, len(instructions))
	}

	for _, instruction := range instructions {
		if instruction == nil {
			t.Fatal("expected rendered instruction, got nil")
		}
		if instruction.ID == nodeJoinOptionalIPCommandID {
			t.Fatalf("instruction list should not include %q", nodeJoinOptionalIPCommandID)
		}
	}
}
